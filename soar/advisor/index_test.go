/*
 * Copyright 2018 Xiaomi, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package advisor

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/XiaoMi/soar/common"
	"github.com/XiaoMi/soar/database"
	"github.com/XiaoMi/soar/env"

	"github.com/kr/pretty"
	"vitess.io/vitess/go/vt/sqlparser"
)

var update = flag.Bool("update", false, "update .golden files")
var vEnv *env.VirtualEnv
var rEnv *database.Connector

func TestMain(m *testing.M) {
	// 初始化 init
	if common.DevPath == "" {
		_, file, _, _ := runtime.Caller(0)
		common.DevPath, _ = filepath.Abs(filepath.Dir(filepath.Join(file, ".."+string(filepath.Separator))))
	}
	common.BaseDir = common.DevPath
	err := common.ParseConfig("")
	common.LogIfError(err, "init ParseConfig")
	common.Log.Debug("advisor_test init")
	vEnv, rEnv = env.BuildEnv()
	if _, err = vEnv.Version(); err != nil {
		fmt.Println(err.Error(), ", By pass all advisor test cases")
		os.Exit(0)
	}

	if _, err := rEnv.Version(); err != nil {
		fmt.Println(err.Error(), ", By pass all advisor test cases")
		os.Exit(0)
	}

	// 分割线
	flag.Parse()
	m.Run()

	// 环境清理
	vEnv.CleanUp()
}

// ARG.003
func TestRuleImplicitConversion(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	dsn := common.Config.OnlineDSN
	common.Config.OnlineDSN = common.Config.TestDSN

	initSQLs := []string{
		`CREATE TABLE t1 (id int, title varchar(255) CHARSET utf8 COLLATE utf8_general_ci);`,
		`CREATE TABLE t2 (id int, title varchar(255) CHARSET utf8mb4);`,
		`CREATE TABLE t3 (id int, title varchar(255) CHARSET utf8 COLLATE utf8_bin);`,
		`CREATE TABLE t4 (id int, col bit(1));`,
	}
	for _, sql := range initSQLs {
		vEnv.BuildVirtualEnv(rEnv, sql)
	}

	sqls := [][]string{
		{
			"SELECT * FROM t1 WHERE title >= 60;",
			"SELECT * FROM t1, t2 WHERE t1.title = t2.title;",
			"SELECT * FROM t1, t3 WHERE t1.title = t3.title;",
			"SELECT * FROM t1 WHERE title in (60, '60');",
			"SELECT * FROM t1 WHERE title in (60);",
			"SELECT * FROM t1 WHERE title in (60, 60);",
			"SELECT * FROM t4 WHERE col = '1'",
		},
		{
			// https://github.com/XiaoMi/soar/issues/151
			"SELECT * FROM t4 WHERE col = 1",
			"SELECT * FROM sakila.film WHERE rental_rate > 1",
		},
	}
	for _, sql := range sqls[0] {
		stmt, syntaxErr := sqlparser.Parse(sql)
		if syntaxErr != nil {
			common.Log.Critical("Syntax Error: %v, SQL: %s", syntaxErr, sql)
		}

		q := &Query4Audit{Query: sql, Stmt: stmt}

		idxAdvisor, err := NewAdvisor(vEnv, *rEnv, *q)
		if err != nil {
			t.Error("NewAdvisor Error: ", err, "SQL: ", sql)
		}

		if idxAdvisor != nil {
			rule := idxAdvisor.RuleImplicitConversion()
			if rule.Item != "ARG.003" {
				t.Error("Rule not match:", rule, "Expect : ARG.003, SQL:", sql)
			}
		}
	}
	for _, sql := range sqls[1] {
		stmt, syntaxErr := sqlparser.Parse(sql)
		if syntaxErr != nil {
			common.Log.Critical("Syntax Error: %v, SQL: %s", syntaxErr, sql)
		}

		q := &Query4Audit{Query: sql, Stmt: stmt}

		idxAdvisor, err := NewAdvisor(vEnv, *rEnv, *q)
		if err != nil {
			t.Error("NewAdvisor Error: ", err, "SQL: ", sql)
		}

		if idxAdvisor != nil {
			rule := idxAdvisor.RuleImplicitConversion()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule, "Expect : OK, SQL:", sql)
			}
		}
	}

	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
	common.Config.OnlineDSN = dsn
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// JOI.003 & JOI.004
func TestRuleImpossibleOuterJoin(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`select city_id, city, country from city left outer join country using(country_id) WHERE city.city_id=59 and country.country='Algeria'`,
		`select city_id, city, country from city left outer join country using(country_id) WHERE country.country='Algeria'`,
		`select city_id, city, country from city left outer join country on city.country_id=country.country_id WHERE city.city_id IS NULL`,
	}

	for _, sql := range sqls {
		stmt, syntaxErr := sqlparser.Parse(sql)
		if syntaxErr != nil {
			common.Log.Critical("Syntax Error: %v, SQL: %s", syntaxErr, sql)
		}

		q := &Query4Audit{Query: sql, Stmt: stmt}

		if vEnv.BuildVirtualEnv(rEnv, q.Query) {
			idxAdvisor, err := NewAdvisor(vEnv, *rEnv, *q)
			if err != nil {
				t.Error("NewAdvisor Error: ", err, "SQL: ", sql)
			}

			if idxAdvisor != nil {
				rule := idxAdvisor.RuleImpossibleOuterJoin()
				if rule.Item != "JOI.003" && rule.Item != "JOI.004" {
					t.Error("Rule not match:", rule, "Expect : JOI.003 || JOI.004")
				}
			}
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// GRP.001
func TestIndexAdvisorRuleGroupByConst(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			`select film_id, title from film where release_year='2006' group by release_year`,
			`select film_id, title from film where release_year in ('2006') group by release_year`,
		},
		{
			// 反面的例子
			`select film_id, title from film where release_year in ('2006', '2007') group by release_year`,
		},
	}

	for _, sql := range sqls[0] {
		stmt, syntaxErr := sqlparser.Parse(sql)
		if syntaxErr != nil {
			common.Log.Critical("Syntax Error: %v, SQL: %s", syntaxErr, sql)
		}

		q := &Query4Audit{Query: sql, Stmt: stmt}

		if vEnv.BuildVirtualEnv(rEnv, q.Query) {
			idxAdvisor, err := NewAdvisor(vEnv, *rEnv, *q)
			if err != nil {
				t.Error("NewAdvisor Error: ", err, "SQL: ", sql)
			}

			if idxAdvisor != nil {
				rule := idxAdvisor.RuleGroupByConst()
				if rule.Item != "GRP.001" {
					t.Error("Rule not match:", rule, "Expect : GRP.001")
				}
			}
		}
	}

	for _, sql := range sqls[1] {
		stmt, syntaxErr := sqlparser.Parse(sql)
		if syntaxErr != nil {
			common.Log.Critical("Syntax Error: %v, SQL: %s", syntaxErr, sql)
		}

		q := &Query4Audit{Query: sql, Stmt: stmt}

		if vEnv.BuildVirtualEnv(rEnv, q.Query) {
			idxAdvisor, err := NewAdvisor(vEnv, *rEnv, *q)
			if err != nil {
				t.Error("NewAdvisor Error: ", err, "SQL: ", sql)
			}

			if idxAdvisor != nil {
				rule := idxAdvisor.RuleGroupByConst()
				if rule.Item != "OK" {
					t.Error("Rule not match:", rule, "Expect : OK")
				}
			}
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// CLA.005
func TestIndexAdvisorRuleOrderByConst(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			`select film_id, title from film where release_year='2006' order by release_year;`,
			`select film_id, title from film where release_year in ('2006') order by release_year;`,
		},
		{
			// 反面的例子
			`select film_id, title from film where release_year in ('2006', '2007') order by release_year;`,
		},
	}

	for _, sql := range sqls[0] {
		stmt, syntaxErr := sqlparser.Parse(sql)
		if syntaxErr != nil {
			common.Log.Critical("Syntax Error: %v, SQL: %s", syntaxErr, sql)
		}

		q := &Query4Audit{Query: sql, Stmt: stmt}

		if vEnv.BuildVirtualEnv(rEnv, q.Query) {
			idxAdvisor, err := NewAdvisor(vEnv, *rEnv, *q)
			if err != nil {
				t.Error("NewAdvisor Error: ", err, "SQL: ", sql)
			}

			if idxAdvisor != nil {
				rule := idxAdvisor.RuleOrderByConst()
				if rule.Item != "CLA.005" {
					t.Error("Rule not match:", rule, "Expect : CLA.005")
				}
			}
		}
	}

	for _, sql := range sqls[1] {
		stmt, syntaxErr := sqlparser.Parse(sql)
		if syntaxErr != nil {
			common.Log.Critical("Syntax Error: %v, SQL: %s", syntaxErr, sql)
		}

		q := &Query4Audit{Query: sql, Stmt: stmt}

		if vEnv.BuildVirtualEnv(rEnv, q.Query) {
			idxAdvisor, err := NewAdvisor(vEnv, *rEnv, *q)
			if err != nil {
				t.Error("NewAdvisor Error: ", err, "SQL: ", sql)
			}

			if idxAdvisor != nil {
				rule := idxAdvisor.RuleOrderByConst()
				if rule.Item != "OK" {
					t.Error("Rule not match:", rule, "Expect : OK")
				}
			}
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// CLA.016
func TestRuleUpdatePrimaryKey(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			`update film set film_id = 1 where title='a';`,
		},
		{
			// 反面的例子
			`select film_id, title from film where release_year in ('2006', '2007') order by release_year;`,
		},
	}

	for _, sql := range sqls[0] {
		stmt, syntaxErr := sqlparser.Parse(sql)
		if syntaxErr != nil {
			common.Log.Critical("Syntax Error: %v, SQL: %s", syntaxErr, sql)
		}

		q := &Query4Audit{Query: sql, Stmt: stmt}

		if vEnv.BuildVirtualEnv(rEnv, q.Query) {
			idxAdvisor, err := NewAdvisor(vEnv, *rEnv, *q)
			if err != nil {
				t.Error("NewAdvisor Error: ", err, "SQL: ", sql)
			}

			if idxAdvisor != nil {
				rule := idxAdvisor.RuleUpdatePrimaryKey()
				if rule.Item != "CLA.016" {
					t.Error("Rule not match:", rule.Item, "Expect : CLA.016")
				}
			}
		}
	}

	for _, sql := range sqls[1] {
		stmt, syntaxErr := sqlparser.Parse(sql)
		if syntaxErr != nil {
			common.Log.Critical("Syntax Error: %v, SQL: %s", syntaxErr, sql)
		}

		q := &Query4Audit{Query: sql, Stmt: stmt}

		if vEnv.BuildVirtualEnv(rEnv, q.Query) {
			idxAdvisor, err := NewAdvisor(vEnv, *rEnv, *q)
			if err != nil {
				t.Error("NewAdvisor Error: ", err, "SQL: ", sql)
			}

			if idxAdvisor != nil {
				rule := idxAdvisor.RuleUpdatePrimaryKey()
				if rule.Item != "OK" {
					t.Error("Rule not match:", rule, "Expect : OK")
				}
			}
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestIndexAdvise(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	orgMinCardinality := common.Config.MinCardinality
	common.Config.MinCardinality = 20

	for _, sql := range common.TestSQLs {
		stmt, syntaxErr := sqlparser.Parse(sql)
		if syntaxErr != nil {
			common.Log.Critical("Syntax Error: %v, SQL: %s", syntaxErr, sql)
		}

		q := &Query4Audit{Query: sql, Stmt: stmt}

		if vEnv.BuildVirtualEnv(rEnv, q.Query) {
			idxAdvisor, err := NewAdvisor(vEnv, *rEnv, *q)
			if err != nil {
				t.Error("NewAdvisor Error: ", err, "SQL: ", sql)
			}

			if idxAdvisor != nil {
				rule := idxAdvisor.IndexAdvise().Format()
				if len(rule) > 0 {
					_, _ = pretty.Println(rule)
				}
			}
		}
	}
	common.Config.MinCardinality = orgMinCardinality
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestIndexAdviseNoEnv(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	orgOnlineDSNStatus := common.Config.OnlineDSN.Disable
	common.Config.OnlineDSN.Disable = true

	for _, sql := range common.TestSQLs {
		stmt, syntaxErr := sqlparser.Parse(sql)
		if syntaxErr != nil {
			common.Log.Critical("Syntax Error: %v, SQL: %s", syntaxErr, sql)
		}

		q := &Query4Audit{Query: sql, Stmt: stmt}

		if vEnv.BuildVirtualEnv(rEnv, q.Query) {
			idxAdvisor, err := NewAdvisor(vEnv, *rEnv, *q)
			if err != nil {
				t.Error("NewAdvisor Error: ", err, "SQL: ", sql)
			}

			if idxAdvisor != nil {
				rule := idxAdvisor.IndexAdvise().Format()
				if len(rule) > 0 {
					pretty.Println(rule)
				}
			}
		}
	}
	common.Config.OnlineDSN.Disable = orgOnlineDSNStatus
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestDuplicateKeyChecker(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	rule := DuplicateKeyChecker(rEnv, "sakila")
	if len(rule) != 0 {
		t.Errorf("got rules: %s", pretty.Sprint(rule))
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestMergeAdvices(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	dst := []IndexInfo{
		{
			Name:     "test",
			Database: "db",
			Table:    "tb",
			ColumnDetails: []*common.Column{
				{
					Name: "test",
				},
			},
		},
	}

	src := dst[0]

	advise := mergeAdvices(dst, src)
	if len(advise) != 1 {
		t.Error(pretty.Sprint(advise))
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestIdxColsTypeCheck(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`select city_id, city, country from city left outer join country using(country_id) WHERE city.city_id=59 and country.country='Algeria'`,
	}

	for _, sql := range sqls {
		stmt, syntaxErr := sqlparser.Parse(sql)
		if syntaxErr != nil {
			common.Log.Critical("Syntax Error: %v, SQL: %s", syntaxErr, sql)
		}

		q := &Query4Audit{Query: sql, Stmt: stmt}

		if vEnv.BuildVirtualEnv(rEnv, q.Query) {
			idxAdvisor, err := NewAdvisor(vEnv, *rEnv, *q)
			if err != nil {
				t.Error("NewAdvisor Error: ", err, "SQL: ", sql)
			}

			idxList := []IndexInfo{
				{
					Name:     "idx_fk_country_id",
					Database: "sakila",
					Table:    "city",
					ColumnDetails: []*common.Column{
						{
							Name:      "country_id",
							Character: "utf8",
							DataType:  "varchar(3000)",
						},
					},
				},
			}

			if idxAdvisor != nil {
				rule := idxAdvisor.idxColsTypeCheck(idxList)
				if !(len(rule) > 0 && rule[0].DDL == "alter table `sakila`.`city` add index `idx_country_id` (`country_id`(N))") {
					t.Error(pretty.Sprint(rule))
				}
			}
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestGetRandomIndexSuffix(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	for i := 0; i < 5; i++ {
		r := getRandomIndexSuffix()
		if !(strings.HasPrefix(r, "_") && len(r) == 5) {
			t.Errorf("getRandomIndexSuffix should return a string with prefix `_` and 5 length, but got:%s", r)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}
