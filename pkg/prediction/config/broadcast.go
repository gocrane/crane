package config

type broadcast struct {
	c chan broadcast
	v interface{}
}

type Broadcaster struct {
	listenCh chan chan (chan broadcast)
	sendCh   chan<- interface{}
}

type Receiver struct {
	c chan broadcast
}

func NewBroadcaster() Broadcaster {
	listenCh := make(chan (chan (chan broadcast)))
	sendCh := make(chan interface{})
	go func() {
		currc := make(chan broadcast, 1)
		for {
			select {
			case v := <-sendCh:
				if v == nil {
					currc <- broadcast{}
					return
				}
				c := make(chan broadcast, 1)
				b := broadcast{c: c, v: v}
				currc <- b
				currc = c
			case r := <-listenCh:
				r <- currc
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
	c := make(chan chan broadcast, 0)
	b.listenCh <- c
	return Receiver{<-c}
}

// Write broadcasts a value to all listeners.
func (b Broadcaster) Write(v interface{}) { b.sendCh <- v }

// Read reads a value that has been broadcast, waiting until one is available if necessary.
func (r *Receiver) Read() interface{} {
	b := <-r.c
	v := b.v
	r.c <- b
	r.c = b.c
	return v
}
