# 常见问题

## 软件依赖

* [git](https://git-scm.co) 项目代码管理工具
* [go](https://golang.org/) 源码编译依赖
* [govendor](https://github.com/kardianos/govendor) 管理第三方包
* [docker](https://www.docker.com) 主要用于构建测试环境
* [mysql](https://www.mysql.com/) 测试时用来连接测试环境
* [retool](https://github.com/twitchtv/retool): 管理测试开发工具,首次安装耗时会比较长,如: `gometalinter.v2`, `revive`, `golangci-lint`

## Web 界面支持

官方不会提供 Web 界面支持，但社区已经有相当多的同学基于 SOAR 开发了衍生的 Web 平台。可以参考如下 ISSUE 的讨论。

> https://github.com/XiaoMi/soar/issues/51

## 命令行参数 `test-dsn`, `online-dsn` 中包含特殊字符怎么办？

如果 `test-dsn` 或 `online-dsn` 中包含':', '@', '/', '!'等特殊字符建议在配置文件中配置相关信息，配置文件为YAML格式，需要遵守YAML格式的要求规范。

## Windows 环境如下安装使用？

SOAR是命令行工具，开源版本无UI界面需要在 `cmd.exe` 下运行。下图为社区同学帮忙录制的 Windows 环境下载、安装及使用演示。

![SOAR__Windows_Installation](https://wx4.sinaimg.cn/large/7143d93fly1fx9z0lw0k8g211j0jlkjo.gif)

## Windows 环境下 -query 文件找不到问题

![](https://raw.githubusercontent.com/XiaoMi/soar/master/doc/images/windows_query_error.png)

将 Window 文件后缀名显示打开，并检查文件名是否正确。

![](https://raw.githubusercontent.com/XiaoMi/soar/master/doc/images/windows_query_check.png)

## Windows环境下双击`soar.windows-amd64`文件无反应。

`soar` 是命令行工具，不是图形化桌面工具，Windows环境需要在 `cmd.exe` 下以命令行方式运行。使用 `soar` 前您需要先熟悉Windows命令行使用。

## Windows 用户的引号问题

![windows_quote](https://raw.githubusercontent.com/XiaoMi/soar/master/doc/images/windows_quote.png)

从上图可以看出 Windows 环境下引号（单引号或双引号）也成为了SQL的一部分传递给 soar 进行分析。因此 Windows 环境下使用如下方式读取 SQL 来解决 Windows 的引号问题。

```bash
type query.sql | soar.windows-amd64
```

## 文件无法执行

```bash
./soar.linux-amd64
bash: ./soar.linux-amd64: cannot execute binary file
```

请注意您操作系统类型，`soar.linux-amd64` 为 Linux 系统使用的二进制文件，`soar.darwin-amd64` 为苹果系统使用的二进制文件，`soar.windows-amd64` 是微软用户使用的二进制文件。下载文件后 Linux 和苹果用户需要为文件添加可执行权限 `chmod a+x filename`。

## 命令无法找到

```bash
bash: soar: command not found
```

直接执行 `soar` 命令提示命令无法找到，请先将 soar 文件添加可执行权限 `chmod a+x soar` 然后将可以将 soar 所在路径加到[PATH](https://linuxconfig.org/linux-path-environment-variable)中，也可以将 soar 移动到已有 PATH 中。

当然在 Linux 环境下，在 soar 二进制文件所在路径运行 `./soar` 也同样可以解决您的问题。

## 提示语法错误

* 请检查SQL语句中是否出现了不配对的引号,如 `, ", '

## 输出结果返回慢

* 如果配置了 online-dsn 或 test-dsn SOAR 会请求这些数据库以支持更多的功能，这时评审一条SQL就会耗时变长。
* 如果又开启了 `-sampling=true` 的话会将线上的数据导入到测试环境，数据采样也会消耗一些时间。

## 如何搭建测试环境

```bash
# 创建测试数据库
wget http://downloads.mysql.com/doc/sakila-db.tar.gz
tar zxf sakila-db.tar.gz && cd sakila-db
mysql -u root -p -f < sakila-schema.sql
mysql -u root -p -f < sakila-data.sql

# 创建测试用户
CREATE USER root@'hostname' IDENTIFIED BY "1t'sB1g3rt";
GRANT ALL ON  *.* TO root@'hostname';
```

## 更新vitess依赖

使用`govendor fetch`或`git clone` [vitess](https://github.com/vitessio/vitess) 在某些地区更新vitess可能会比较慢，导致项目编译不过，所以将vitess整个代码库加到了代码仓库。

如属更新vitess仓库可以使用如下命令。

```bash
make vitess
```

## 生成报告并发邮件

```bash
#!/bin/bash

soar -query "select * from film" > ./index.html

(
  echo To: youmail@example.com
  echo From: robot@example.com
  echo "Content-Type: text/html; "
  echo Subject: SQL Analyze Report
  echo
  cat ./index.html
) | sendmail -t

```

## 如何新增一条启发式建议

```bash
advisor/rules.go HeuristicRules 加一个条新的规则
advisor/heuristic.go 实现一个规则函数
advisor/heuristic_test.go 添加相应规则函数的测试用例
make heuristic
make daily
```
