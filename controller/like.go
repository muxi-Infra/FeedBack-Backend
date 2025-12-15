package controller

import (
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/api/request"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"github.com/muxi-Infra/FeedBack-Backend/service"

	"github.com/gin-gonic/gin"
)

type Like struct {
	l        service.LikeService
	tableCfg *config.AppTable
	log      logger.Logger
}

func NewLike(l service.LikeService, tableCfg *config.AppTable, log logger.Logger) *Like {
	var like = &Like{
		l:        l,
		tableCfg: tableCfg,
		log:      log,
	}
	go like.HandleLikeTask()
	go like.MoveRetry2Pending()

	return like
}

// AddLikeTask 添加点赞任务
// @Summary 添加点赞任务
// @Description 添加一个点赞或取消点赞的任务
// @Tags 点赞
// @Accept json
// @Produce json
// @Param request body request.LikeReq true "点赞请求参数"
// @Success 200 {object} response.Response "添加点赞任务成功"
// @Failure 400 {object} response.Response "添加点赞任务失败"
// @Router /like/addtask [post]
func (l *Like) AddLikeTask(c *gin.Context, r request.LikeReq, uc ijwt.UserClaims) (response.Response, error) {
	table := l.tableCfg.Tables[uc.NormalTableID]
	err := l.l.AddLikeTask(l.tableCfg.AppToken, table.TableID, r.RecordID, r.UserID, r.IsLike, r.Action)
	if err != nil {
		return response.Response{
			Code:    400,
			Message: "添加点赞任务失败",
		}, err
	}
	return response.Response{
		Code:    200,
		Message: "添加点赞任务成功",
	}, nil
}

func (l *Like) HandleLikeTask() {
	go func() {
		for {
			l.l.HandleLikeTask() // 这里是有阻塞的
			time.Sleep(time.Millisecond * 100)
		}
	}()
}

func (l *Like) MoveRetry2Pending() {
	go func() {
		for {
			err := l.l.MoveRetry2Pending()
			if err != nil {
				l.log.Error("MoveRetry2Pending failed",
					logger.String("error", err.Error()),
				)
			}
			time.Sleep(time.Millisecond * 100)
		}
	}()
}
