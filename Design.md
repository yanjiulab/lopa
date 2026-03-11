# Lopa Network Measurement Tool Design Documentation

## 1. Overview

Lopa is a **lightweight, centralized, single-binary deployed** network quality measurement and monitoring tool. Its core features are simplicity, stability, no dependencies, and easy scalability. It focuses on measuring key network indicators such as latency, packet loss, and jitter, and supports unified management of distributed multi-nodes. It will be written in pure Go. 

## 2. Supported Protocols

Lopa focuses on practical network quality measurement and supports the following core protocols:

- PING: Based on ICMP protocol

- TCPING: Based on TCP protocol, measures the connectivity and latency of specific TCP ports (supports port specification).

- UDP Probe: Based on UDP protocol, measures latency, packet loss, and jitter of UDP packets (supports custom packet size).

- TWAMP: Follow the standard RFC

- TWAMP-light: Follow the standard RFC

All protocols share the same task and result structure, and the measurement logic is uniformly managed by Lopa. Reflector does not distinguish between protocols and only reflects packets in their original form.

## 3. Core Architecture

### 3.1 Lopa (Master/Measurement Engine)

- Responsible for **packet transmission, timing, statistics, calculation, result generation, task management, and interface provision**.
- All measurement logic, protocol processing, and data statistics are concentrated in Lopa.
- No dependence on databases, middleware, or external services; runs purely in memory.

### 3.2 Lopa Reflector (Reflection Ability)

- Only performs **pure packet reflection**; returns packets immediately upon receipt.

- Stateless, no calculation, no storage, and no business parsing.

- The simpler it is, the more stable it is; can be deployed in large quantities across platforms and network segments.

- Acts as a "remote echo point" for Lopa to implement cross-machine and cross-segment measurement.

## 4. Measurement Modes

Lopa supports three standard measurement modes, all using a **unified task structure and unified result structure**:

### 4.1 Fixed Packet Count Mode (count)

- Automatically ends after sending a specified number of probe packets.

- Suitable for short-term testing, quick verification, and batch detection.

- Outputs final statistics: packet loss rate, average/minimum/maximum latency, jitter.

### 4.2 Fixed Duration Mode (duration)

- Continuously sends packets within a specified time and ends automatically when the time is up.

- Suitable for network quality evaluation within a period of time.

- Outputs total statistics for the entire period.

### 4.3 Continuous Measurement Mode (continuous)

- Continuously sends packets until manually stopped by the user.

- Suitable for **long-term network monitoring and stability observation**.

- **Total packet loss rate is not used** (to avoid contamination by historical packets); only **sliding window statistics** are used.

- The window reflects the current network status in real time, avoiding the problem of "permanently non-zero packet loss rate after one packet loss".

## 5. Measurement Round Mechanism

To more accurately observe network fluctuations, Lopa supports **multi-round measurement**:

- Each round is counted independently.

- An interval can be configured between rounds.

- Results retain detailed data of each round, facilitating the observation of intermittent packet loss and sudden latency increases.

- Continuous mode (continuous) does not use rounds and uses sliding windows instead.

## 6. Task ID Design

To support **multi-node and unified central control display**, the task ID adopts the following format:

**Unique Node Identifier + Incremental Sequence in the Node**

Example: `192.168.1.10-123`

- Globally unique, no conflicts.

- Central control can directly distinguish tasks from different nodes.

- Simple, readable, and easy to retrieve.

## 7. Measurement Task Parameters (Unified Structure)

All measurements use the same set of parameters, including:

- Measurement type (ping/tcp/udp)

- Target address, protocol version (ipv4/ipv6)

- Packet interval, timeout time, packet size

- Source IP, network card specification

- Measurement mode (count/duration/continuous)

- Number of packets (for count mode), duration (for duration mode)

- Number of rounds, interval between rounds

- Continuous mode only: packet loss threshold, latency threshold, alert callback URL

## 8. Measurement Result Structure (Unified Structure)

All measurements output the same result format, including:

- Basic task information (ID, target, node, time)

- Total statistics (number of sent packets, number of received packets, packet loss rate, latency statistics)

- Detailed multi-round data (independent data for each round)

- Continuous mode only: **real-time sliding window data** (window packet loss, window latency)

- Error information, execution status

## 9. CLI Design 

The command line interface (CLI) is the **primary user interaction method** of Lopa, designed to be simple, intuitive, and easy to use. All operations can be completed with simple commands, and no complex configuration is required.

All commands follow the format: `lopa [command] [subcommand] [options]`. 

Using cobra as cli implementation. it is the primary user interaction method, supporting task initiation, real-time result viewing, and task management.

CLI Output Description

- Fixed mode (count/duration): After the task ends, output the final statistics directly (task ID, target, packet loss rate, latency, etc.).

- Continuous mode (continuous): Real-time refresh of sliding window data (updated every 10 seconds by default), showing the current network status.

- Error prompt: If the task fails (e.g., target unreachable), output a clear error message.

## 10. HTTP REST API 

- Create measurement tasks

- Query task status

- Query measurement results

- Stop tasks

- Unified aggregation of multi-nodes

## 11. Continuous Measurement Monitoring Capabilities

Only the continuous mode supports monitoring and alerting capabilities:

- **Sliding Window Statistics**: Network quality in the last N seconds, real-time, clean, and not affected by history.

- **Abnormal Threshold Alert**: Triggered when the packet loss rate exceeds the threshold or the latency exceeds the threshold.

- **HTTP Callback Notification (WebHook)**: Users provide a simple HTTP interface; Lopa automatically pushes alert JSON when an abnormality is detected. Users can connect to DingTalk, Lark, log systems, or monitoring platforms by themselves.

One-time measurement (count/duration) **does not support alerting** and only outputs final results.

## 12. Ways for Users to Obtain Measurement Results

1. **CLI Instant Output**: One-time tasks print complete results after completion; continuous tasks refresh sliding windows in real time.
2. **HTTP API Query**: Query historical results and real-time status by task ID; unified display on the central control interface.
3. **Continuous Task Alert Push**: Automatic callback for abnormalities, no need for active polling.

## 13. Implementation details

project module name: github.com/yanjiulab/lopa

executable file: lopa

lopa use these libs:

- cobra & viper: API and command line
- echo: http
- zap: log