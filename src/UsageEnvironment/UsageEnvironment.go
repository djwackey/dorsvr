package UsageEnvironment

type BasicTaskScheduler struct {
}

type BasicUsageEnvironment struct {
}

func UsageEnvironment() *BasicUsageEnvironment {
	return new(BasicUsageEnvironment)
}

func TaskScheduler() *BasicTaskScheduler {
	return new(BasicTaskScheduler)
}

func (scheduler *BasicTaskScheduler) DoEventLoop() {
	select {}
}
