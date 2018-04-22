 Netlink
===========
dump linux kernel network statistics through `netlink` protocol

Scenario
----
  * 在超高并发服务器上, 通过 linux 内核 **netlink** 协议直接获取主机上所有连接状态的统计数, 从而避免了扫描 `/proc` 统计的低效耗时
