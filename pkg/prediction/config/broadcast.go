package config

type event struct {
	nextEventCh chan event
	value       interface{}
}

type Broadcaster struct {
	listenCh chan chan (chan event)
	sendCh   chan<- interface{}
}

type Receiver struct {
	ch chan event
}

func NewBroadcaster() Broadcaster {
	listenCh := make(chan (chan (chan event)))
	sendCh := make(chan interface{})
	go func() {
		eventCh := make(chan event, 1)
		for {
			select {
			case v := <-sendCh:
				if v == nil {
					eventCh <- event{}
					return
				}
				nextEventCh := make(chan event, 1)
				// Put the event into the event channel so that one of the receiver can read it.
				eventCh <- event{nextEventCh: nextEventCh, value: v}
				eventCh = nextEventCh
			case recvCh := <-listenCh:
				// Send the event channel to the receiver that requests to listen.
				// If an event is put into the event channel, only one of the receiver can read it.
				// But it doesn't matter, since the receiver who has read the event will put it back
				// into the receiver channel and forget it.
				recvCh <- eventCh
			}
		}
	}()
	return Broadcaster{
		listenCh: listenCh,
		sendCh:   sendCh,
	}
}

// Listen starts listening to the broadcasts.
func (b Broadcaster) Listen() Receiver {
	ch := make(chan chan event, 0)
	b.listenCh <- ch
	return Receiver{<-ch}
}

// Write broadcasts a value to all listeners.
func (b Broadcaster) Write(v interface{}) { b.sendCh <- v }

// Read reads a value that has been broadcast, waiting until one is available if necessary.
func (r *Receiver) Read() interface{} {
	e := <-r.ch
	v := e.value
	// Put the event back to the receivers' channel, so that one of the other receiver can read it.
	r.ch <- e
	r.ch = e.nextEventCh
	return v
}
