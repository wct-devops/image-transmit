## WINDOWS下的容器镜像传输工具

各团队在给项目发布版本的时候，都会涉及到将容器镜像从公司的官方仓库复制到项目的仓库上，通常我们会在局方找一台Linux作为跳板机，安装好Docker工具，然后docker pull/docker tag/docker push的方式来进行，
这台Linux主机需要有单独申请访问外网的权限，而通常提供的跳板机都是Windows机器，由于Docker镜像占用空间较多，还需要定时的清理或者维护跳板机，否则容易出现各种问题。

目前我们版本升级都通过界面实现了一键操作的傻瓜方式，而传输镜像因为涉及网络权限和跳板机，很多还是手工操作。因此开发了这个Windows的镜像传输工具方便大家使用。

工具的优势有：
- 绿色版，无需安装任何三方工具，下载exe，然后参考样例写一个cfg.yaml配置文件即可运行
- 全部内存操作，不占用本地存储，对机器要求比较低，传输效率高
- Windows下的界面操作，不懂Docker命令的小白也可以搞定
- 并发数支持可配置，可以根据网络条件随时调整

## 配置文件
在工具的相同目录下,放置一个配置文件cfg.yaml,其内容参考如下：
```yaml
source: # 源仓库信息配置,可以支持多个
- registry: "http://10.45.80.1"
  user: #用户名和密码，如果匿名访问，留空即可
  password:
target:  # 目标仓库信息配置,可以支持多个
- registry: "http://10.45.46.109"
  user: 
  password: 
  #repository: # 可选配置，是否修改镜像名称，假如填写值yyyy，则会将源仓库的10.45.80.1/xxxx/image:tag统一改成10.45.46.109/yyyy/image:tag
#maxconn: 2 # 可选配置，最大并发数
#retries: 1 # 可选配置，最大重试次数
```

## 界面截图
![image](https://user-images.githubusercontent.com/11539396/118996182-49507f80-b9ba-11eb-81c6-97d20facdf38.png)

## 使用说明
1. 选择源仓库和目标仓库，按需调整并发度和异常重试次数
2. 在左侧输入框输入需要传输的镜像列表，会自动忽略仓库URL地址等信息，统一使用选择的源仓库的URL地址
3. 可以点击校验用来校验一下输入的镜像列表信息是否正确
4. 点击【开始】按钮，启动镜像复制，界面会自动刷新日志和实时统计
5. 用户可以点击【停止】中断镜像的传输
6. 目前只提供Windows X64版本


## 致谢
使用到的开源库:  
https://github.com/AliyunContainerService/image-syncer  
https://github.com/lxn/walk

碰到问题欢迎大家提issue
