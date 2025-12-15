package service

import "github.com/muxi-Infra/FeedBack-Backend/repository/dao"

type SheetService interface {
	GetUserLikeRecord(recordID string, userID string) (int, error)
}

type SheetServiceImpl struct {
	likeDao dao.Like
}

func NewSheetService(likeDao dao.Like) SheetService {
	return &SheetServiceImpl{
		likeDao: likeDao,
	}
}

func (s *SheetServiceImpl) GetUserLikeRecord(recordID string, userID string) (int, error) {
	return s.likeDao.GetUserLikeRecord(recordID, userID)
}
