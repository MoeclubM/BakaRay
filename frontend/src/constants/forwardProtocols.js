export const directProtocolOptions = [
  { title: "TCP", value: "tcp", description: "标准 TCP 单点转发" },
  { title: "UDP", value: "udp", description: "标准 UDP 单点转发" }
]

export const tunnelProtocolOptions = [
  { title: "TLS", value: "tls", description: "通过 TLS 承载隧道流量" },
  { title: "mTLS", value: "mtls", description: "通过多路复用 TLS 承载隧道流量" },
  { title: "WS", value: "ws", description: "通过 WebSocket 承载隧道流量" },
  { title: "MWS", value: "mws", description: "通过多路复用 WebSocket 承载隧道流量" },
  { title: "WSS", value: "wss", description: "通过加密 WebSocket 承载隧道流量" },
  { title: "MWSS", value: "mwss", description: "通过多路复用加密 WebSocket 承载隧道流量" },
  { title: "gRPC", value: "grpc", description: "通过 gRPC 承载隧道流量" },
  { title: "H2", value: "h2", description: "通过 HTTP/2 承载隧道流量" },
  { title: "H2C", value: "h2c", description: "通过明文 HTTP/2 承载隧道流量" },
  { title: "KCP", value: "kcp", description: "通过 KCP 承载隧道流量" },
  { title: "QUIC", value: "quic", description: "通过 QUIC 承载隧道流量" }
]

export const nodeCapabilityOptions = [
  ...directProtocolOptions,
  ...tunnelProtocolOptions
]

const protocolMap = Object.fromEntries(
  nodeCapabilityOptions.map((item) => [item.value, item])
)

export function getForwardProtocolTitle(protocol) {
  return protocolMap[protocol]?.title || protocol || "-"
}

export function getForwardProtocolDescription(protocol) {
  return protocolMap[protocol]?.description || ""
}
