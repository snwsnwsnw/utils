package wsutil

import (
	"fmt"
	"github.com/google/uuid"
)

// TrackLog 發送 websocket 時，填寫TrackLog資料以追蹤消息 格式[TrackIdent][TrackIUuid]
type TrackLog struct {
	// TrackIdent 建議填入可識別的字串
	TrackIdent string
	// TrackIUuid 建議你填入uuid以供識別 實際上填入啥都行 留空會自動產生uuid
	TrackIUuid string
}

func (l TrackLog) Log() string {
	return fmt.Sprintf(
		"[%s][%s]",
		l.TrackIdent,
		func() string {
			if l.TrackIUuid != "" {
				return l.TrackIUuid
			}
			return uuid.NewString()
		}())
}
