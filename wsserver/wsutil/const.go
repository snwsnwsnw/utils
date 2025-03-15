package wsutil

const (
	// direct 收到direct的訊息會直接將內容廣播出去
	WebSocketKeyDirect = "direct"
	// 指定會員
	WebSocketKeySpecificPlayerId      = "specificPlayerId"      // 指定會員ID
	WebSocketKeySpecificChannelPlayer = "specificChannelPlayer" // 會員 透過ChannelId分組
	WebSocketKeySpecificPackagePlayer = "specificPackagePlayer" // 會員 透過PackageId分組
	WebSocketKeySpecificAgentPlayer   = "specificAgentPlayer"   // 會員 透過AgentId分組

	WebSocketKeySpecificAgentId = "specificAgentId"   // 所有連線(會員+訪客) 透過AgentId分組
	WebSocketKeySpecificPackage = "specificPackageId" // 所有連線(會員+訪客) 透過PackageId分組
	WebSocketKeySpecificChannel = "specificChannel"   // 所有連線(會員+訪客) 透過ChannelId分組

	WebSocketKeySpecificChannelVisitor = "specificChannelVisitor" // 訪客 透過ChannelId分組
	WebSocketKeySpecificPackageVisitor = "specificPackageVisitor" // 訪客 透過PackageId分組
	WebSocketKeySpecificAgentVisitor   = "specificAgentVisitor"   // 訪客 透過AgentId分組

	WebSocketKeySpecificPlayerIdKick = "specificPlayerIdKick"
)
