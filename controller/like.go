package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/muxi-Infra/FeedBack-Backend/api/request"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"github.com/muxi-Infra/FeedBack-Backend/service"
)

type Like struct {
	l   service.LikeService
	log logger.Logger
}

func NewLike(l service.LikeService, log logger.Logger) *Like {
	return &Like{
		l:   l,
		log: log,
	}
}

// AddLikeTask 添加点赞任务
//
//	@Summary		添加点赞任务
//	@Description	添加一个点赞或取消点赞的任务
//	@Tags			点赞
//	@Accept			json
//	@Produce		json
//	@Param			request	body		request.LikeReq		true	"点赞请求参数"
//	@Success		200		{object}	response.Response	"添加点赞任务成功"
//	@Failure		400		{object}	response.Response	"添加点赞任务失败"
//	@Router			/like/addtask [post]
func (l *Like) AddLikeTask(c *gin.Context, r request.LikeReq, uc ijwt.UserClaims) (response.Response, error) {
	// 组装参数
	//record := DTO.TableRecord{
	//	Record: r.Record,
	//}
	//tableConfig := DTO.TableConfig{
	//	TableName:  &uc.TableName,
	//	TableToken: &uc.TableToken,
	//	TableID:    &uc.TableId,
	//	ViewID:     &uc.ViewId,
	//}

	err := l.l.AddLikeTask(uc.TableToken, uc.TableId, r.RecordID, r.UserID, r.IsLike, r.Action)
	if err != nil {
		return response.Response{}, err
	}
	return response.Response{
		Code:    200,
		Message: "添加点赞任务成功",
	}, nil
}
