# ISP Context

ISP 上下文描述一次连接中的对端身份，以及巡视业务中的设备点位身份。连接身份、业务对端身份和设备身份必须保持独立。

## Language

**Session ID**:
一次 TCP 连接的临时身份；同一对端重连后会获得新的 Session ID。
_Avoid_: Client ID, Device Code

**Client ID**:
对端完成协议注册后绑定的稳定身份；ISP 管理接口将它称为 Device Code。
_Avoid_: Session ID, Device ID

**Send Code**:
ISP 消息所声明的发送端身份。
_Avoid_: Session ID

**Receive Code**:
ISP 消息所声明的接收端身份。
_Avoid_: Client ID

**Device ID**:
巡视模型中设备点位的身份，而不是连接或协议对端的身份。
_Avoid_: Client ID, Session ID
