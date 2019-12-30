
# 体系架构

![架构图](https://raw.githubusercontent.com/XiaoMi/soar/master/doc/images/structure.png)

SOAR主要由语法解析器，集成环境，优化建议，重写逻辑，工具集五大模块组成。下面将对每个模块的作用及设计实现进行简述，更详细的算法及逻辑会在各个独立章节中详细讲解。

## 语法解析和语法检查

一条SQL从文件，标准输入或命令行参数等形式传递给SOAR后首先进入语法解析器，这里一开始我们选用了vitess的语法解析库作为SOAR的语法解析库，但随时需求的不断增加我们发现有些复杂需求使用vitess的语法解析实现起来比较逻辑比较复杂。于是参考业务其他数据库产品，我们引入了TiDB的语法解析器做为补充。我们发现这两个解析库还存在一定的盲区，于是又引入了MySQL执行返回结果作为多版本SQL方言的补充。大家也可以看到在语法解析器这里，SOAR的实现方案是松散的、可插拔的。SOAR并不直接维护庞大的语法解析库，它把各种优秀的语法解析库集成在一起，各取所长。

## 集成环境

集成环境区分`线上环境`和`测试环境`两种，分别用于解决不同场景下用户的SQL优化需求。一种常见的情况是已有表结构需要优化查询SQL的场景，可以从线上环境导出表结构和足够的采样数据到测试环境，在测试环境上就可以放心的执行各种高危操作而不用担心数据被损坏。另一种常见的情况是建一套全新的数据库，需要验证提供的数据字典中是否存在优化的可能。对于这种情况，很有可能你不需要知道线上环境在哪儿，完全只是想先试试看，如果报错了马上改对就是了。当然还有更多种组合的场景需求，将在[集成环境](http://github.com/XiaoMi/soar/blob/master/doc/environment.md)中介绍。

## 优化建议

目前SOAR可以提供的优化建议有基于启发式规则(通常也称之为经验)的优化建议，基于索引优化算法给出的索引优化建议，以及基于EXPLAIN信息给出的解读。

### 启发式规则建议

下面这段代码是启发式规则的的元数据结构，它由规则代号，危险等级，规则摘要，规则解释，SQL示例，建议位置，规则函数等7部分组成。每一条SQL经过语法解析后会经过数百个启发式规则的逐一检查，命中了的规则将会保存在一个叫heuristicSuggest的变量中传递下去，与其他优化建议合并输出。这里最核心的部分，也是代码最多的部分在heuristic.go，里面包含了所有的启发式规则实现的函数。所有的启发式规则列表保存在rules.go文件中。

```Golang
// Rule 评审规则元数据结构
type Rule struct {
    Item     string                  `json:"Item"`     // 规则代号
    Severity string                  `json:"Severity"` // 危险等级：L[0-8], 数字越大表示级别越高
    Summary  string                  `json:"Summary"`  // 规则摘要
    Content  string                  `json:"Content"`  // 规则解释
    Case     string                  `json:"Case"`     // SQL示例
    Position int                     `json:"Position"` // 建议所处SQL字符位置，默认0表示全局建议
    Func     func(*Query4Audit) Rule `json:"-"`        // 函数名
}
```

### 索引优化

关于索引优化，数据库经过几十年的发展，DBA沉淀了很多宝贵的经验，怎样把这些感性的经验转化为覆盖全面、逻辑可推导的算法是这种模块最大的挑战。很幸运的是SOAR并不是第一个尝试做这类算法整理的产品，有很多前人的著作、论文、博客等的知识储备。毫不夸张的说，为了写成这个模块我们读了不下5百万字的著作和论文，还不包括网络上各种大神的博客，这些老师们的知识结晶收集整理在[鸣谢](http://github.com/XiaoMi/soar/blob/master/doc/thanks.md)章节。使用到的算法在[索引优化](http://github.com/XiaoMi/soar/blob/master/doc/indexing.md)章节有详细的描述，虽然在某些算法理解上可能还存在一定争议，很希望与同行们共同讨论，共同进步，不断完善SOAR的算法。

### EXPLAIN解读

做过SQL优化的人对EXPLAIN应该都不陌生，但对于新手来说要记住每一个列代表什么含义，每个关键字背后的奥秘是什么需要足够的脑容量来记忆才行。统计了一下SOAR只在EXPLAIN信息的注解一项差不多写了200行代码，按平均行长度120计算，算下来一个DBA要精通EXPLAIN优化就要记住不下2万字的文档。SOAR能帮每为DBA节约了这部分脑容量。不过关于EXPLAIN解读还远不止这些，想了解更多可以参考[EXPLAIN信息解读](http://github.com/XiaoMi/soar/blob/master/doc/explain.md)章节。

## 重写逻辑

上面提到的优化建议是我们早期实现的主要功能，早期的功能还只是停留在建议上，对于一些初级用户看到建议也不一定会改写。为了进一步简化SQL优化的成本，SOAR又进一步挖掘了自动SQL重写的功能。现在提供几十种常见场景下的SQL等价转写，不过相比SQL优化建议还有很大的改进空间。这部分的功能和逻辑将在[重写逻辑](http://github.com/XiaoMi/soar/blob/master/doc/rewrite.md)一章中详细说明。

## 工具集

除了SQL优化和改写以外，为了方便用户使用以及美化输出展现形式，SOAR还提供了一些辅助的小工具，比如markdown转HTML工具，SQL格式化输出工具等等。你可以在[常用命令](http://github.com/XiaoMi/soar/blob/master/doc/cheatsheet.md)中找到这些小工具的使用方法。
