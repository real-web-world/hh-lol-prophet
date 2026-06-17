# lol 对局先知
qq 群: 102158075

网站: [https://lol.buffge.com](https://lol.buffge.com)

[下载地址](https://lol.buffge.com)

### 程序逻辑
   - 监控lol client
     - 存在   -> 开始监听游戏事件
     - 不存在 -> 关闭游戏事件监视器 
### 游戏事件监听器
   - 监听lol事件
   - 如果进入英雄选择阶段 则进入分析程序

### 分析程序:
- 获取队伍所有用户信息
  - 查询每个用户最近20局战绩并计算得分
- 根据所有用户的分数判断小代上等马中等马下等马
- 发送到选人界面

## 特性
- 自动更新
- 自动接受对局
- 自动ban pick
- 查询用户马匹信息

## lcu代理模式

http://localhost:4396/v1/lcu/proxy 为lcu代理接口 可调用所有接口也可连接lcu wss,

前端用户可直接在网页开发业务功能,可玩性极高

## [计分规则](./计分方式.md)

## 开发计划
- 优化算法
  - 根据对位数据差 计分
  - 对特定位置 计算特定指标 如 对打野计算参团率 如低于50% 扣分
  - 服务端
    -上报计算数据 每一局 每个人kda 实际得分
  - gui
    - 有gui后考虑加上有趣的功能
    
- 优化lol.buffge.com网站
- 配置:
   - 遇到特定用户 发送特定消息 比如 "遇霸不秒退,十五两行累" "西内!" "小心家猪野猪"
  
## Code signing policy
- Free code signing provided by [SignPath.io](https://about.signpath.io/), certificate by [SignPath Foundation](https://signpath.org/)
- Team roles and their members
  - Committers and reviewers：[Members team](https://github.com/orgs/real-web-world/teams/members)
  -  Approvers：[Owners](https://github.com/orgs/real-web-world/people?query=role%3Aowner)
- [Privacy policy](https://lol.buffge.com/privacyPolicy.html)

# Sponsors

感谢以下赞助商 :heart:

| ![signpath](https://signpath.org/assets/favicon-50x50.png) | Free code signing on Windows provided by [SignPath.io](https://about.signpath.io/),certficate by [SignPath Foundation](https://signpath.org/) |
|------------------------------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------|
                                                                |

## 🔋 JetBrains 开源证书支持

`hh-lol-prophet` 项目一直以来都是在 JetBrains 公司旗下的 GoLand 集成开发环境中进行开发，基于 **free JetBrains Open Source license(s)** 正版免费授权，在此表达我的谢意。

<a href="https://www.jetbrains.com/?from=hh-lol-prophet" target="_blank"><img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/jetbrains/jetbrains-variant-4.png" width="250" align="middle"/></a>  


