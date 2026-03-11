# TWAMP-light 实现评估与兼容标准 Reflector 的工作项

## 1. 协议要点（RFC 5357 Appendix I + RFC 4656）

- **TWAMP-light**：无 TCP 控制连接，Session-Sender 与 Session-Reflector 通过本地配置，直接使用 TWAMP-Test 包格式在 UDP 上通信。
- **标准端口**：TWAMP-Test 使用 **UDP 862**（与 TWAMP-Control 的 TCP 862 不同；Light 仅用 UDP 862）。
- **Session-Sender 发包格式（RFC 4656 §4.1.2）**  
  - Sequence Number：4 字节，大端。  
  - Timestamp：8 字节，大端；高 4 字节为整数秒（since 1970），低 4 字节为秒的小数部分（单位 1/2^32 秒）。  
  - Error Estimate：2 字节。  
  - MBZ：2 字节，必须为 0。  
  - 以上共 **16 字节**，之后可填充 0 以达到所需包长。
- **Session-Reflector 回包格式（RFC 5357 §4.2.1）**  
  - 前 16 字节与发送包相同：Sequence Number、T1（发送方时间戳）、Error Estimate、MBZ。  
  - 随后为 Reflector 追加的块：  
    - T2（Reflector 收到时间）：8 字节时间戳 + 2 字节 Error + 2 字节 MBZ；  
    - T3（Reflector 发送时间）：8 字节时间戳 + 2 字节 Error + 2 字节 MBZ。  
  - 回包最小长度 **16 + 12 + 12 = 40 字节**（可再填充）。
- **兼容性**：与“其他标准 reflector”兼容，只需：  
  - 客户端按上述格式发送 Session-Sender 包到 `target:862`；  
  - 能解析 40 字节以上的 Reflector 回包（Seq、T1、T2、T3 等），并用 T1 与本地收发时间计算 RTT（或可选地使用 T2/T3 做更细分析）。

## 2. 需完成的工作清单

| 序号 | 工作项 | 说明 |
|------|--------|------|
| 1 | **TWAMP 包格式与编解码** | 实现发送包 16 字节头 + padding；解析回包 40 字节（Seq、T1、T2、T3、Error、MBZ）。时间戳按 RFC：32b 整秒 + 32b 小数。 |
| 2 | **TWAMP-light 客户端（Session-Sender）** | 实现 `protocol.TWAMPPinger`：构造并发送 RFC 格式包，接收回包，按 Sequence Number 匹配，用发送时间与接收时间计算 RTT，实现 `Pinger` 接口。 |
| 3 | **引擎与 API/CLI** | 增加 `CreateTwampTask`、`runTwamp`；`POST /api/v1/tasks/twamp`；CLI `lopa twamp <target>`，默认端口 862。 |
| 4 | **可选：内置 Reflector 支持 TWAMP** | 当前内置 Reflector 为“原样回显”。若需与标准 reflector 一致，可增加在 **862** 上监听、识别 TWAMP 包（长度≥16）、按 RFC 填 T2/T3 并回 40 字节格式；否则仅实现客户端即可兼容“其他标准 reflector”。 |

## 3. 最小实现范围（兼容标准 reflector）

- **必须**：完成 1、2、3（客户端 + 包格式 + 引擎/API/CLI），使 Lopa 作为 Session-Sender 可向任意标准 TWAMP-light Session-Reflector（如第三方设备或软件）发起测量。
- **可选**：完成 4，使 Lopa 自带的 Reflector 在 862 上也可作为标准 TWAMP-light reflector，便于自测或与其它标准实现互测。

## 4. 与现有 UDP Probe 的差异

- **UDP Probe**：自定义 8 字节 seq + padding，对端为“原样回显”的 Reflector（如当前 lopad 的 8081）。  
- **TWAMP-light**：RFC 规定的 16 字节头 + 可选 padding，时间戳格式固定；对端为标准 TWAMP-light Session-Reflector（通常 862），回包为 40+ 字节的 T1/T2/T3 格式。  
两者互不替代：TWAMP-light 用于与标准 reflector 互通，UDP Probe 用于与 Lopa 自带 Reflector 或其它“只回显”的实现互通。

## 5. 已实现内容（本仓库）

- **TWAMP-light 客户端**：`internal/protocol/twamp_light.go`，按 RFC 发送 16 字节头 + padding，解析 40 字节回包，按 Sequence 匹配并计算 RTT；默认目标端口 862。
- **引擎 / API / CLI**：`CreateTwampTask`、`runTwamp`；`POST /api/v1/tasks/twamp`；`lopa twamp <target>`，支持 `--port 862`、`--size`（最小 16）等。
- **可选 TWAMP-light Reflector**：`internal/reflector/twamp.go`，在 `reflector.twamp_addr`（默认 `:862`）上响应标准 40 字节格式，便于自测或与其它标准实现互测。通过 `LOPA_REFLECTOR_TWAMP_ADDR=` 置空可关闭。
