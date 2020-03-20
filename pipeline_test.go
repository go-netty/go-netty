package netty

import "testing"

type oneHandler struct{}

func (h oneHandler) HandleActive(ctx ActiveContext) {}

type twoHandler struct{}

func (h twoHandler) HandleRead(ctx InboundContext, message Message) {}

type threeHandler struct{}

func (h threeHandler) HandleWrite(ctx OutboundContext, message Message) {}

type fourHandler struct{}

func (h fourHandler) HandleException(ctx ExceptionContext, ex Exception) {}

type fiveHandler struct{}

func (h fiveHandler) HandleInactive(ctx InactiveContext, ex Exception) {}

func TestPipeline(t *testing.T) {

	pipeline := NewPipelineWith()

	if 2 != pipeline.Size() {
		t.Fatal("headHandler / tailHandler")
	}

	pipeline.AddLast(twoHandler{})
	pipeline.AddFirst(oneHandler{})

	twoIndex := pipeline.IndexOf(func(handler Handler) bool {
		_, ok := handler.(twoHandler)
		return ok
	})

	if 2 != twoIndex {
		t.Fatal("twoHandler:", twoIndex)
	}

	pipeline.AddHandler(twoIndex, threeHandler{}, fourHandler{}, fiveHandler{})

	for i := 1; i < pipeline.Size()-1; i++ {
		handler := pipeline.ContextAt(i).Handler()
		switch handler.(type) {
		case oneHandler:
			if 1 != i {
				t.Fatal("unexpect position: ", i, "want: ", 1)
			}
		case twoHandler:
			if 2 != i {
				t.Fatal("unexpect position: ", i, "want: ", 2)
			}
		case threeHandler:
			if 3 != i {
				t.Fatal("unexpect position: ", i, "want: ", 3)
			}
		case fourHandler:
			if 4 != i {
				t.Fatal("unexpect position: ", i, "want: ", 4)
			}
		case fiveHandler:
			if 5 != i {
				t.Fatal("unexpect position: ", i, "want: ", 5)
			}
		default:
			t.Fatal("invalid handler")
		}
	}

}
