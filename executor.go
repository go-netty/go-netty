package netty

type Action = func()

type Executor interface {
	Exec(Action)
}

func AsyncExecutor() Executor {
	return asyncExecutor{}
}

type asyncExecutor struct{}

func (asyncExecutor) Exec(action Action) { go action() }
