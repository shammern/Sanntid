package elevator

import (
	"elevator-project/pkg/drivers"
	"fmt"
	"sync"
	"time"
)

// OrderQueue represents a queue for handling button events
type OrderQueue struct {
	Queue         []drivers.ButtonEvent
	Mutex         sync.Mutex
	AddChan       chan drivers.ButtonEvent
	NextOrder     chan int
	ElevatorDone  chan bool
	ElevatorIdle  chan bool // Signals when the elevator is idle and ready
}

// NewOrderQueue initializes an empty OrderQueue
func NewOrderQueue() *OrderQueue {
	q := &OrderQueue{
		Queue:        []drivers.ButtonEvent{},
		AddChan:      make(chan drivers.ButtonEvent, 10),
		NextOrder:    make(chan int, 1),
		ElevatorDone: make(chan bool, 1),
		ElevatorIdle: make(chan bool, 1), // To notify when FSM is idle
	}
	go q.queueWorker()
	go q.queuePrinter()
	go q.processNextOrder()
	return q
}

func (q *OrderQueue) queueWorker() {
	for event := range q.AddChan {
		q.Mutex.Lock()
		wasEmpty := len(q.Queue) == 0 // Correct place to check if empty
		q.Queue = append(q.Queue, event)
		q.Mutex.Unlock()

		// If the queue was empty, signal FSM to start immediately
		if wasEmpty {
			fmt.Println("Queue was empty, sending order immediately")
			q.sendNextOrder()
		}
	}
}

// processNextOrder waits for the FSM to signal completion before sending the next order
func (q *OrderQueue) processNextOrder() {
	for {
		select {
		case <-q.ElevatorDone: // FSM signals completion
			fmt.Println("FSM finished an order, checking next.")
			q.sendNextOrder()
		case <-q.ElevatorIdle: // FSM signals idle state
			fmt.Println("FSM is idle, checking queue for new orders.")
			q.Mutex.Lock()
			if len(q.Queue) > 0 {
				q.Mutex.Unlock()
				q.sendNextOrder()
			} else {
				q.Mutex.Unlock()
			}
		}
	}
}


func (q *OrderQueue) sendNextOrder() {
	q.Mutex.Lock()
	defer q.Mutex.Unlock()

	fmt.Println("sendNextOrder triggered. Queue size before sending:", len(q.Queue))

	if len(q.Queue) > 0 {
		order := q.Queue[0]
		q.Queue = q.Queue[1:]
		fmt.Println("Queue sending order to FSM:", order.Floor)
		fmt.Println("Queue size after sending:", len(q.Queue)) // Debugging
		q.NextOrder <- order.Floor
	} else {
		fmt.Println("Queue is empty, FSM is now idle")
		q.ElevatorIdle <- true
	}
}



func (q *OrderQueue) Enqueue(event drivers.ButtonEvent) {
	q.Mutex.Lock()
	defer q.Mutex.Unlock()

	fmt.Println("Enqueuing order:", event)
	q.Queue = append(q.Queue, event)
	fmt.Println("Current queue size after enqueue:", len(q.Queue)) // Debugging log

	go q.sendNextOrder() // Ensure immediate processing
}



// PrintQueue prints the current state of the queue for debugging
func (q *OrderQueue) PrintQueue() {
	q.Mutex.Lock()
	defer q.Mutex.Unlock()
	fmt.Println("Current Queue:")
	for i, event := range q.Queue {
		fmt.Printf("%d: Floor %d, Button %d\n", i, event.Floor, event.Button)
	}
}

// queuePrinter prints the queue every 5 seconds (debugging only)
func (q *OrderQueue) queuePrinter() {
	for {
		time.Sleep(5 * time.Second)
		q.PrintQueue()
	}
}
