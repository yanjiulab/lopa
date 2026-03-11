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

---

# 在 TWAMP-light 之上增加完整 TWAMP 协议支持

## 1. TWAMP 与 TWAMP-light 的差异

| 项目 | TWAMP-light（已实现） | 完整 TWAMP（RFC 5357） |
|------|------------------------|------------------------|
| 控制通道 | 无，两端本地配置 | **TWAMP-Control**：TCP 862，协商会话参数 |
| 测试通道 | 直接向配置好的 reflector 地址:端口发 UDP | 与 light 相同（TWAMP-Test），但 **目标地址与端口由控制协商得出** |
| 会话建立 | 无 | Request-TW-Session → Accept-Session → Start-Sessions |

即：在现有 TWAMP-light 客户端之上，增加「先通过 TCP 862 做控制握手、拿到 reflector 的地址与端口，再按同样格式发 TWAMP-Test 包」的能力，即支持完整 TWAMP。

## 2. 需要新增的工作

### 2.1 TWAMP-Control 客户端（必须）

- **TCP 连接**：对 `server:862` 建立 TCP（与现有 UDP 862 测试端口区分：同一端口号，协议不同）。
- **连接建立（RFC 5357 §3.1）**  
  - 收 **Server Greeting**：Unused(12)、Modes(4)、Challenge(16)、Salt(16)、Count(4)、MBZ(12)。  
  - 发 **Set-Up-Response**：Mode(4)、KeyID(80)、Token(64)、Client-IV(16)。  
  - **非认证模式（Mode=1）**：KeyID/Token/Client-IV 可填 0，实现最简单，先做此模式即可。
- **Server-Start**：收 Server 的 Accept、Server-IV、Start-Time、MBZ。
- **创建会话（§3.5 Request-TW-Session）**  
  - 发 Command Number=5，Conf-Sender=0、Conf-Receiver=0，Number of Scheduled Slots=0、Number of Packets=0。  
  - **Sender Port**：本机用于发/收 TWAMP-Test 的 UDP 端口（可由系统分配，再填入）。  
  - **Receiver Port**：Reflector 侧接收/发送测试包的 UDP 端口（通常 862，或由 Server 在 Accept-Session 里确认/建议）。  
  - **Sender Address / Receiver Address**：本机与 Reflector 的 IP；可填 0 表示「用当前控制连接的源/目的 IP」。  
  - SID=0、Start Time、Timeout、Type-P、MBZ、HMAC（非认证时 HMAC 为 0）。
- **Accept-Session**：收 Server 的 Accept、Port（确认或建议的 Receiver Port）、HMAC 等。
- **Start-Sessions**：发 Start-Sessions 命令，Server 确认后即可开始发 TWAMP-Test 包。
- **（可选）Stop-Sessions**：结束时发 Stop-Sessions，便于 Server 回收资源。

实现时建议单独包或文件，例如 `internal/protocol/twamp_control.go`，提供：`Dial(serverAddr) → Session{ReflectorAddr, ReflectorPort, SenderPort}`，供上层在「完整 TWAMP」模式下使用。

### 2.2 复用现有 TWAMP-Test 逻辑

- **不重复造轮子**：TWAMP-Test 包格式、时间戳编解码、40 字节回包解析已 in `twamp_light.go`。  
- **完整 TWAMP 的 Session-Sender**：  
  - 先通过 TWAMP-Control 拿到 **Reflector 地址与端口**（及本机 Sender Port，若需绑定）。  
  - 再使用与 TWAMP-light 相同的发包/收包与 RTT 计算逻辑，仅把「目标地址:端口」改为协商得到的 reflector 地址与端口。  
- 可选做法：  
  - 在 `TWAMPPinger` 上增加「从外部注入 Target（reflector addr:port）」的构造方式，由「完整 TWAMP」流程在控制握手后注入；或  
  - 新增 `TWAMPSession`：内部持有一个 `TWAMPPinger`（或相同发包/解析逻辑），在 `Start()` 时先跑控制握手，再按协商结果创建 Pinger 并执行 count/duration/continuous 测量。

### 2.3 引擎与 API/CLI 的扩展

- **任务类型或模式**：  
  - 保留现有 `twamp`（TWAMP-light）：目标为「直接配置的 reflector 地址:端口」，无控制连接。  
  - 新增「完整 TWAMP」模式：目标为「TWAMP Server 地址」（如 `host` 或 `host:862`），先 TCP 控制握手，再对协商出的 reflector 发 TWAMP-Test。  
- **参数**：  
  - 完整 TWAMP 需指定「Server 地址」、可选「本机发送端口」；其余（count/duration/interval/timeout/size 等）与 light 一致。  
- **引擎**：  
  - 例如 `CreateTwampTask(params)` 根据 `params` 某字段（如 `UseControl: bool` 或 `Mode: "light"|"full"`）决定：  
    - light：沿用现有 `TWAMPPinger` + 给定 Target；  
    - full：先 `twamp_control.Dial` 拿到会话，再对协商出的 reflector 跑同一套 Test 逻辑。  
- **API**：  
  - 现有 `POST /api/v1/tasks/twamp` 可扩展 body 增加 `use_control` 或 `mode`，或新增 `POST /api/v1/tasks/twamp-full`。  
- **CLI**：  
  - 例如 `lopa twamp <target> --control` 或 `lopa twamp-full <server>`，表示对 `<server>` 做完整 TWAMP（先控制，再测试）。

### 2.4 认证与加密（可选，后续）

- RFC 5357 支持 **Mode 2（认证）**、**Mode 4（加密）**：需实现 HMAC、AES、密钥派生等。  
- 建议先只做 **Mode 1（非认证）**，与多数仅需内网/测试场景的 Server 兼容；认证/加密作为后续迭代。

### 2.5 服务端（Lopa 作为 TWAMP Server + Reflector）（可选）

- 若希望 Lopa 不仅做 Reflector，还做 **TWAMP Server**（接受控制连接并管理会话）：  
  - 实现 **TWAMP-Control Server**：TCP 862 监听、Greeting、处理 Set-Up-Response、Request-TW-Session、Accept-Session、Start-Sessions、Stop-Sessions。  
  - 与现有「仅 UDP 862 的 TWAMP-light Reflector」可共存：同一主机上 TCP 862 给控制、UDP 862 给测试；或由 Server 在 Accept-Session 中指定其它测试端口。  
- 此为较大独立功能，可与「仅做 Control-Client + 复用现有 Test」分开实现。

## 3. 建议实现顺序

1. **TWAMP-Control 客户端（仅 Mode 1）**：TCP 连接、Greeting/Set-Up-Response/Server-Start、Request-TW-Session、Accept-Session、Start-Sessions；解析出 Reflector 地址与端口。  
2. **在引擎中增加「完整 TWAMP」分支**：根据参数先执行控制握手，再使用现有 TWAMP-Test 逻辑对协商出的目标测 RTT。  
3. **API 与 CLI**：扩展或新增接口与子命令，传入 Server 地址与 `--control`（或等价格式）。  
4. （可选）Stop-Sessions、认证/加密、Lopa 端 Server 实现。

## 4. 小结

在现有 **TWAMP-light**（无控制、直接对 reflector 发 UDP 测试包）之上增加 **完整 TWAMP** 支持，核心是：  
- 实现 **TWAMP-Control 客户端**（TCP 862，至少 Mode 1）；  
- 用控制协商得到的 **Reflector 地址与端口** 驱动现有 **TWAMP-Test** 发包与 RTT 计算；  
- 在引擎/API/CLI 中增加「完整 TWAMP」模式并传入 Server 地址。  
测试包格式、时间戳、40 字节回包解析均可复用，无需重写。
