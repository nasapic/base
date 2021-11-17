package base

type (
	Service interface {
		Worker
	}
)

type (
	BaseService struct {
		*BaseWorker
	}
)

func NewService(name string, log Logger) *BaseService {
	return &BaseService{
		BaseWorker: NewWorker(name, log),
	}
}
