## 增强的旁路cpuset管理能力
kubelet支持static的cpu manager策略，当guaranteed pod运行在节点上时，kebelet会为该pod分配指定的专属cpu，其他进程无法占用，这保证了guaranteed pod的cpu独占，但是也造成了cpu和节点的的利用率较低，造成了一定的浪费。
crane agent为cpuset管理提供了新的策略，允许pod和其他pod共享cpu当其指定了cpu绑核时，可以在利用绑核更少的上下文切换和更高的缓存亲和性的优点的前提下，还能让其他workload部署共用，提升资源利用率。

1. 提供了3种pod cpuset类型：

- exclusive：绑核后其他container不能再使用该cpu，独占cpu
- share：绑核后其他container可以使用该cpu
- none：选择没有被exclusive pod的container占用的cpu，可以使用share类型的绑核

  share类型的绑核策略可以在利用绑核更少的上下文切换和更高的缓存亲和性的优点的前提下，还能让其他workload部署共用，提升资源利用率

2. 放宽了kubelet中绑核的限制

   原先需要所有container的CPU limit与CPU request相等 ，这里只需要任意container的CPU limit大于或等于1且等于CPU request即可为该container设置绑核


3. 支持在pod运行过程中修改pod的 cpuset policy，会立即生效

   pod的cpu manager policy从none转换到share，从exclusive转换到share，均无需重启

使用方法：
1. 设置kubelet的cpuset manager为"none"
2. 通过pod annotation设置cpu manager policy

   `qos.gocrane.io/cpu-manager: none/exclusive/share`
   ```yaml
   apiVersion: v1
   kind: Pod
   metadata:
     annotations:
       qos.gocrane.io/cpu-manager: none/exclusive/share
   ```