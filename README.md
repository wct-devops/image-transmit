# 容器镜像传输工具

## 支持WINDOWS下使用
各团队在给项目发布版本的时候，都会涉及到将容器镜像从公司的官方仓库复制到项目的仓库上，一般我们需要让客户找一台Linux作为跳板机，安装好Docker工具，然后docker pull/docker tag/docker push的方式来进行，这台Linux主机需要有单独申请访问外网的权限。这个申请流程比较麻烦，而更常见的是客户会提供一个Windows跳板机。在Windows上安装Docker比较麻烦，而且由于Docker镜像占用空间较多，还需要定时的清理或者维护跳板机，否则容易出现各种异常。

目前我们版本升级都通过界面实现了一键操作的傻瓜方式，而传输镜像因为涉及网络权限和跳板机，很多还是手工操作。因此开发了这个Windows的镜像传输工具方便大家使用。

工具的优势有：
- 绿色版，无需安装任何三方工具，直接下载一个EXE程序(大小仅5M)到主机，参考样例写一个cfg.yaml配置文件即可运行
- 最小化占用CPU和存储资源，对机器要求比较低，传输效率高
- Windows下的界面操作，不懂Docker命令的小白也可以搞定
- 并发数和重试数支持可配置，可以根据网络条件调整
- 同时支持离线模式和直传模式

### 配置文件
在工具的目录下,放置一个配置文件cfg.yaml(或者data目录下),其内容参考如下：
```yaml
source: # 源仓库信息配置,可以支持多个
  registry: "http://10.45.80.1"
  user: #用户名和密码，如果匿名访问，用户名和密码都留空即可
  password:
  #name: #可选配置,指定名称
target:  # 目标仓库信息配置,可以支持多个
  registry: "http://10.45.46.109"
  user: 
  password: 
  #repository: # 可选配置，是否修改镜像名称，假如填写值yyyy，则会将源仓库的10.45.80.1/xxxx/image:tag统一改成10.45.46.109/yyyy/image:tag
  #name: #可选配置,指定名称
#maxconn: 5 # 可选配置，最大并发数，默认5
#retries: 2 # 可选配置，最大重试次数，默认2
#singlefile: false #可选配置，是否生成单一文件，默认关
#compressor: # 可选配置。如果不配置，windows下默认为tar模式, linux下如果系统存在mksquashfs/tar,且运行时为特权账号(root或者sudo)，则采用squashfs模式，否则为tar模式，详细解释参考说明
#lang: en_US # 可选配置，指定语言版本,支持中英文两种语言，默认取操作系统语言
#cache:   # 可选配置，是否开启本地缓存，默认关，详细参考说明
#outprefix: # 可选配置，用于指定生成的压缩文件的前缀，也可以在执行命令时使用-out参数来指定
#  pathname: cache # 缓存目录
#  keepdays: 7  # 缓存最长保留时间，默认7
#  keepsize: 10  # 缓存目录最大使用量，单位G，默认10
```

### 界面截图
![image](https://user-images.githubusercontent.com/11539396/121972464-c3073d80-cdad-11eb-8067-ac1d26cba791.png)

### 使用说明
#### 直传场景
假设客户提供的跳板机可以连接公网上的仓库，也可以同时连接内网的仓库，则优先推荐使用这个模式。直传模式逐个从源仓库拉取镜像，然后直接推动到目标仓库，文件不落地，网络带宽有保障即可，同时只会传输目标仓库上不存在的分层，因此效率是最高的，如果直通网络有保障，这个是最简单的模式。
1. 选择源仓库和目标仓库，按需调整并发度和异常重试次数
2. 在左侧输入框输入需要传输的镜像列表，会自动替换镜像列表URL地址信息，统一使用选择的源仓库的URL地址，无需手工去替换
3. 可以点击校验用来验证一下源镜像列表和目标镜像列表信息匹配是否正确
4. 点击【直传】按钮，启动镜像复制，界面会自动刷新日志和实时统计
5. 用户可以点击【停止】中断镜像的传输

> 直传模式下是否启用本地缓存的说明  
> 通常不需要启用本地缓存。启用本地缓存后，每次向目标传输的文件都会在缓存目录中保存一份，下次传输时会优先使用缓存的包。有两种场景使用本地缓存可以有所帮助。
> 1. 每次需要向多个目标仓库同步相同的镜像，使用本地缓存，一些包只需要从源仓库下载一次
> 2. 跳板机网络不稳定，使用缓存可以尽量减少一次重复的包传输

#### 离线场景
一些客户不允许提供可以开放公网访问的跳板机，因此不能使用直传模式，需要使用离线模式，即先在研发中心将镜像打包成文件，然后转移到跳板机，然后从跳板机上将文件上传到目标仓库。整个过程需要使用到【下载】和【上传】两个按钮。
1. 选择源仓库，按需调整并发度和异常重试次数，以及增量或者全量模式，是否使用单一文件模式等
2. 在左侧输入框输入需要传输的镜像列表，会自动替换镜像列表URL地址信息，统一使用选择的源仓库的URL地址，无需手工去替换
3. 可以点击校验用来验证一下源镜像列表信息是否正确
4. 点击【下载】按钮，启动镜像下载，程序会在程序所在的目录下按照【日期】创建一个新文件夹，并且根据时间戳创建对应的数据文件和描述文件，示例如下：
```
├─20210612
│      img_full_202106122344_0.tar
│      img_full_202106122344_1.tar
│      img_full_202106122344_meta.yaml
```
5. 用户将生成的几个文件全部传输给客户，放置到跳板机的某个目录下。
6. 登录跳板机，选择目标仓库，按需调整并发度和异常重试次数，点击【上传】按钮，在弹出文件框中选择上一步中传输过去的描述文件，确认信息后，开始上传
7. 上传时会根据描述文件，对数据文件进行个数/大小的校验，如果报错，请检查数据文件是否传输完整。

> 关于增量和全量的说明  
> 在离线传输模式下，由于无法同时连接源仓库和目标仓库，因此无法判断需要传输哪些镜像分层，全量模式就是将所有的分层都打包到数据文件中，是最安全的方式，但是这种模式下数据文件会比较大。
> 当选择增量模式时，会自动弹出文件框，要求选择一个【上次传输的描述文件】，会根据这个文件将上次传输过的镜像分层都跳过，只打包没有传输过的分层，因此可以大幅降低包的大小。但是这种模式下在上传增量镜像的时候，需要通过流程保证已经将【上次传输的描述文件】的镜像导入到目标库了，不能跳过。
> 一个建议的增量和全量发布流程如下：
> 1. 每个月初发布一个全量的版本
> 2. 每天发布增量版本，弹出框时选择【月初的全量版本】，这样即可只发增量变更的分层
> 3. 现场保证每次的全量版本必须导入，直接增量则导入需要的增量包即可（不需要将每个增量包逐次导入）
> 4. 在_meta.yaml文件有本次版本依赖的版本的信息，可以进行确认，形如`https://last.img/skip/it:img_full_202106130839_meta.yaml`，
> 4. 开发人员临时发送个别镜像测试时，选择全量模式更安全

> 关于单一文件的说明  
> 通常下载时选择使用几个并发，就会产生几个独立数据文件，但是某些场景下，用户希望只需要一个数据文件，方便传输，则可以打开【单一文件】的开关。打开这个开关后，文件都会先被下载到临时目录中，最后被合并成一个文件，因此占用空间和IO，时间也会更长。
> 如果下载时是在同一个局域网内，不存在网络瓶颈，可以在界面上设置并发度为1或者在cfg.yaml中增加一个maxconn=1的配置，只使用一个网络线程，也可以实现只生成一个文件，并且避免了本地操作，可以验证一下两种方式哪种更快。

## 命令行模式
龙舟版开始同时支持windows和linux下命令行方式进行传输：

1. 由于需要通过命令行来指定源/目标仓库信息，因此建议在配置文件中为每个仓库配置一个name，以方便使用。如果不指定，则默认的name为"registry-repository"
2. 使用命令行时, 可以将镜像列表放到一个文本文件，然后通过-lst参数传递给程序，也可以直接使用输入流的方式来指定，简化日常操作，可以参见下文中的使用示意。

```
zoms@172.16.85.48[/home/zoms]$ image-transmit 
Invalid args, please refer the help
Image Transmit-DragonBoat-WhaleCloud DevOps Team
./image-transmit [OPTIONS]
Examples: 
            Save mode:           ./image-transmit -src=nj -lst=img.lst
            Increment save mode: ./image-transmit -src=nj -lst=img.lst -inc=img_full_202106122344_meta.yaml
            Transmit mode:       ./image-transmit -src=nj -lst=img.lst -dst=gz
            Upload mode:           ./image-transmit -dst=gz -img=img_full_202106122344_meta.yaml
More description please refer to github.com/wct-devops/image-transmit
  -dst string
        Destination repository name, default: the first repo in cfg.yaml
  -img string
        Image meta file to upload(*meta.yaml)
  -inc string
        The referred image meta file(*meta.yaml) in increment mode
  -lst string
        Image list file, one image each line
  -src string
        Source repository name, default: the first repo in cfg.yaml
  -out string
        Output filename prefix
zoms@172.16.85.48[/home/zoms]$ image-transmit -src=nj <<EOF
10.45.80.21/public/alpine:3.11
10.45.80.21/public/alpine:3.12.1
10.45.80.21/public/alpine:3.12.2
EOF
[2021-06-14 21:42:12] Get 3 images
[2021-06-14 21:42:12] Create data file: img_full_202106142142_0.tar
[2021-06-14 21:42:12] Create data file: img_full_202106142142_1.tar
[2021-06-14 21:42:14] Generated a download task for 10.45.80.1/public/alpine:3.11
[2021-06-14 21:42:16] Generated a download task for 10.45.80.1/public/alpine:3.12.1
[2021-06-14 21:42:17] Url http://10.45.80.1/public/alpine:3.12.2 format error: Error reading manifest 3.12.2 in 10.45.80.1/public/alpine: manifest unknown: manifest unknown, skipped
[2021-06-14 21:42:17] Start processing taks, total 2 ...
[2021-06-14 21:42:17] Get manifest from 10.45.80.1/public/alpine:3.11
Invalid:1 Total:2 Success:0 Failed:0 Doing:1 Down:0.0B/s Up:0.0B/s, Total Down:0.0B Up:0.0B Time:
[2021-06-14 21:42:17] Get manifest from 10.45.80.1/public/alpine:3.12.1
[2021-06-14 21:42:17] Get a blob sha256:cbdbe7a5bc2a(2.7MB) from 10.45.80.1/public/alpine:3.11 success
[2021-06-14 21:42:17] Get a blob sha256:188c0c94c7c5(2.7MB) from 10.45.80.1/public/alpine:3.12.1 success
[2021-06-14 21:42:17] Get a blob sha256:f70734b6a266(1.5KB) from 10.45.80.1/public/alpine:3.11 success
[2021-06-14 21:42:17] Get a blob sha256:d6e46aa2470d(1.5KB) from 10.45.80.1/public/alpine:3.12.1 success
[2021-06-14 21:42:17] Task completed, total 2 tasks with 0 failed
[2021-06-14 21:42:17] WARNING: there are 1 images failed with invalid url(ex:image not exists)
[2021-06-14 21:42:17] Invalid url list:
http://10.45.80.1/public/alpine:3.12.2
[2021-06-14 21:42:17] Create meta file: data/20210614/img_full_202106142142_meta.yaml
```

```
C:\Users\WangYuMu\go\src\github.com\wct-devops\image-transmit\cmd>image-transmit-cmd.exe
命令行参数不正确，请查看帮助
云雀-镜像传输工具-龙舟版-浩鲸DevOps团队
image-transmit-cmd.exe [选项]
例子:
            下载模式:           image-transmit-cmd.exe -src=nj -lst=img.lst
            增量下载模式:       image-transmit-cmd.exe -src=nj -lst=img.lst -inc=img_full_202106122344_meta.yaml
            直传模式:           image-transmit-cmd.exe -src=nj -lst=img.lst -dst=gz
            上传模式:           image-transmit-cmd.exe -dst=gz -img=img_full_202106122344_meta.yaml
更多帮助可以访问github.com/wct-devops/image-transmit
  -dst string
        目标仓库名称, 默认为第一个
  -img string
        需要上传的镜像规格文件(*meta.yaml)
  -inc string
        指定增量模式下参考的镜像规格文件(*meta.yaml)
  -lst string
        镜像列表文件,一行一个
  -src string
        源仓库名称, 默认为配置文件中的第一个仓库
  -out string
        输出压缩文件的前缀
C:\Users\WangYuMu\go\src\github.com\wct-devops\image-transmit\cmd>(echo 10.45.80.21/public/alpine:3.11 && echo 10.45.80.21/public/alpine:3.12.1 ) | image-transmit-cmd.exe
-src=nj
[2021-06-14 21:52:28] 读取2个镜像
[2021-06-14 21:52:28] 生成数据文件: img_full_202106142152_0.tar
[2021-06-14 21:52:28] 生成数据文件: img_full_202106142152_1.tar
[2021-06-14 21:52:31] 完成10.45.80.1/public/alpine:3.11下载任务创建
[2021-06-14 21:52:35] 完成10.45.80.1/public/alpine:3.12.1下载任务创建
[2021-06-14 21:52:35] 开始处理任务，总计2个，请稍后...
无效:0 总计:2 成功:2 失败:0 正在处理:0 下载速度:0.0B/s 上传速度:0.0B/s 总下载:0.0B 总上传:0.0B 耗时:
[2021-06-14 21:52:35] 从 10.45.80.1/public/alpine:3.11 下载manifest完成
[2021-06-14 21:52:35] 从 10.45.80.1/public/alpine:3.12.1 下载manifest完成
[2021-06-14 21:52:35] 下载blob sha256:188c0c94c7c5(2.7MB) 自 10.45.80.1/public/alpine:3.12.1 完成
[2021-06-14 21:52:35] 下载blob sha256:cbdbe7a5bc2a(2.7MB) 自 10.45.80.1/public/alpine:3.11 完成
无效:0 总计:2 成功:0 失败:0 正在处理:2 下载速度:0.0B/s 上传速度:0.0B/s 总下载:0.0B 总上传:0.0B 耗时:
无效:0 总计:2 成功:0 失败:0 正在处理:2 下载速度:0.0B/s 上传速度:0.0B/s 总下载:0.0B 总上传:0.0B 耗时:
无效:0 总计:2 成功:0 失败:0 正在处理:2 下载速度:0.0B/s 上传速度:0.0B/s 总下载:0.0B 总上传:0.0B 耗时:
[2021-06-14 21:52:39] 下载blob sha256:f70734b6a266(1.5KB) 自 10.45.80.1/public/alpine:3.11 完成
无效:0 总计:2 成功:1 失败:0 正在处理:1 下载速度:549.8KB/s 上传速度:0.0B/s 总下载:2.7MB 总上传:0.0B 耗时:
[2021-06-14 21:52:39] 下载blob sha256:d6e46aa2470d(1.5KB) 自 10.45.80.1/public/alpine:3.12.1 完成
[2021-06-14 21:52:39] 任务处理结束，总计2任务，0任务失败
[2021-06-14 21:52:39] 生成压缩规格文件: data\20210614\img_full_202106142152_meta.yaml
```

> 如何将镜像直接导入到本机的Docker？  
> 通常我们上传的压缩包是直接导入到镜像仓库中，但是有些场景，比如一键部署时，需要将镜像直接导入到本地Docker来使用，这个时候可以直接指定参数-dst=docker就可以将目标设置为本地docker

> 关于不同压缩方式的说明  
> 目前支持两种方式tar和squashfs，两种模式的区别有：  
> 1. tar全程不需要压缩和解压重新处理，因此打包和解包效率非常高，打包时间一般是squashfs的一半
> 2. squashfs需要将容器镜像每一层的包都解开到本地，需要占用大量本地文件系统空间，但是由于squashfs文件系统支持重复文件识别等固实压缩优化，比tar模式节省30%左右大小
> 3. 由于squashfs需要将每一层压缩包都解开到临时目录，并逐个扫描压缩，容器镜像中存在很多系统文件，因此对权限要求比较高，需要使用root账号或者sudo命令来执行，否则会报错
> 4. 目前Golang并没有非常完善的squashfs解压缩和tar解压到文件系统的包，目前测试比较稳定的是在Linux下，使用Linux自带的mksquashfs、tar工具来进行解包和打包，因此windows下不建议使用squashfs，除非你能确认镜像包中只包含普通权限的文件，可以尝试。在windows下压缩需要下载squashfs.zip包并解压到同一目录下
> 5. squashfs的temp目录大小需要足够存放整个镜像未压缩的文件

> 建议使用缓存目录  
> 如果使用一些固定的机器来给项目发布镜像，可以打开缓存，这样可以避免每次重复下载已有的镜像层，大大提高打包的效率

### 版本下载说明
请到[release](https://github.com/wct-devops/image-transmit/releases)页面下载
- image-transmit : Linux命令行版
- image-transmit-cmd.zip : Windows命令行版
- image-transmit-gui.zip : Windows桌面版


### 多语言支持
目前支持中英两种语言，会自动根据操作系统的设置自动切换。如果想强制切换，有两种方式：
1. 在cfg.yaml中通过lang参数来指定en_US/zh_CN
2. 在环境变量中指定lang参数

### 致谢
使用到的开源库:  
https://github.com/AliyunContainerService/image-syncer  
https://github.com/lxn/walk  
https://github.com/klauspost/compress/zstd  
https://github.com/ulikunitz/xz  
https://github.com/pierrec/lz4  
  
碰到问题欢迎大家提issue
