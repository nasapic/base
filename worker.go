package base

type (
	Worker interface {
		Name() string
		Init() bool  // NOTE: Better return an error?
		Start() bool // NOTE: Better return an error?
	}
)

type (
	BaseWorker struct {
		name     string
		didInit  bool
		didStart bool
		log      Logger
	}
)

func NewWorker(name string, log Logger) *BaseWorker {
	name = genName(name, "worker")

	return &BaseWorker{
		name: name,
		log:  log,
	}
}

func (bw BaseWorker) Name() string {
	return bw.name
}

func (bw BaseWorker) SetName(name string) {
	bw.name = name
}

func (bw BaseWorker) Init() bool {
	return bw.didInit
}

func (bw BaseWorker) Start() bool {
	return bw.didStart
}
