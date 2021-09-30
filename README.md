# shiba

shiba（柴犬）真是个可爱的动物

## 服务启动流程
1. 根据配置(配置项log)初始化日志模块
2. 按照优先级从小到大调用模块的Init函数，初始化模块
3. 解码每个模块的配置
4. 按照优先级从小到大调用模块的Start函数，启动模块
5. 开启服务监听
6. 收到信号关闭服务
7. 按照优先级从大到小调用模块的Stop函数，停止模块
8. 结束日志模块
9. 服务推出


## 功能

1. 支持按优先级注册模块
2. 模块配置自动加载解析
3. 多数据库、redis配置
4. 可通过命令行flag和配置文件指定服务的端口
5. 可通过命令行flag指定配置文件路径
6. 可以通过命令行指定日志等级，通过http动态调整日志等级`/log/level`
7. 集成zap日志

## TODO

- [ ] 内部模块有对应配置才进行初始化，外部模块都进行初始化
- [ ] 缓存(支持不同的缓存策略)
- [ ] 通过地址或者本地路径自更新
- [ ] 自守护