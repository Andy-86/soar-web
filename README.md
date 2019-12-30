# soar-web
* 项目是基于soar的sql检查工具，项目地址如下[soar](https://github.com/XiaoMi/soar)。

# 增功能如下
1.新增web功能，本项目基于rest接口方式对sql进行检查，而原有对是每次检查都需要启动一次，这样可以复用数据库连接池，达到更高的效率。
> eg:http://127.0.0.1:9090/http/sql?sql=select id,email  from user where id = 1 and did = 2  and email = '222'   and nick_name = 'aaa'&database=religion（里新增两个参数，一个是sql可以直接传输或者urlencode传入，第二个是databasdatabase：即使用那个数据源。

2.增加多数据源配置，原有项目只支持一个数据源，本项目支持多个数据源
>eg:databasesSource:
   - addr: 127.0.0.1:3306 <br/>
     schema: crawer <br/>
     user: root <br/>
     password: root <br/>
     disable: false <br/>
     alias: public <br/>
   - addr: 127.0.0.1:3306 <br/>
     schema: test <br/>
     user: root <br/>
     password: root <br/>
     disable: false <br/>
     alias: religion <br/>
   - addr: 127.0.0.1:3306 <br/>
     schema: test <br/>
     user: root <br/>
     password: root <br/>
     disable: false <br/>
     alias: assn <br/>
   - addr: 127.0.0.1:3306 <br/>
     schema: test <br/>
     user: root <br/>
     password: root <br/>
     disable: false <br/>
     alias: scenic <br/><br\>
   * alias字段为数据库对别名，与第一部分对数据源传入参数相对应，具体可以参照/etc/soar.yaml
   
3.soar 检查索引是根据索引和where条件对包含关系来判读是否需要新增索引，本项目只针对mysql innodb 改为左前匹配，比如 where a=*
and b = * and c = * ,但是索引是 a,d,e  。这种情况innodb 认为已经走上索引。
