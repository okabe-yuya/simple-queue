package main

import (
	"errors"
	"fmt"
	"time"
)

type MessageKind int

const (
	ENQUEUE = 1
	DEQUEUE = 2
	CHECK_QUEUE_SIZE = 3
	DOWN = 4
)

type ServerSetting struct {
	QueueSize int32;
}

type Message struct {
	kind MessageKind;
	value int32;
	client chan int32;
}

func (s *ServerSetting) run() chan Message {
	ch := make(chan Message)
	queue := make([]int32, 0, s.QueueSize)

	go func(ch chan Message) {
		for {
			select {
			case message := <- ch:
				switch message.kind {
				case ENQUEUE:
					queue = append(queue, message.value)
				case DEQUEUE:
					if len(queue) > 0 {
						head := queue[0]
						queue = queue[1:]

						message.client <- head
					} else {
						message.client <- -1 // これはイケてないが暫定処理
					}
				case CHECK_QUEUE_SIZE:
					message.client <- int32(len(queue))
				case DOWN:
					fmt.Println("safety ShutDown queue server ...")
					close(ch)
					break
				}
			}
		}
	}(ch)

	return ch
}

func (s *ServerSetting) down(server chan Message) error {
	m := createMessage(DOWN, 0, nil)
	server <- *m

	return nil
}

func createMessage(kind MessageKind, value int32, client chan int32) *Message {
	m := &Message{ kind: kind, value: value, client: client }
	return m
}

func Enqueue(server chan Message, value int32, client chan int32 ) error {
	m := createMessage(ENQUEUE, value, client)
	server <- *m

	_, err := m.receive(client)
	if err != nil {
		return err
	}
	return nil
}

func Dequeue(server chan Message, client chan int32) (int32, error) {
	m := createMessage(DEQUEUE, 0, client)
	server <- *m

	v, err := m.receive(client)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func CheckQueueSize(server chan Message, client chan int32) (int32, error) {
	m := createMessage(CHECK_QUEUE_SIZE, 0, client)
	server <- *m

	v, err := m.receive(client)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func (m *Message) receive(client chan int32) (int32, error) {
	switch m.kind {
	case ENQUEUE, DOWN:
		return 0, nil
	case DEQUEUE, CHECK_QUEUE_SIZE:
		v := <-client
		return v, nil
	default:
		return 0, errors.New("サポートされていないメッセージです")
	}
}


func CreateClient() chan int32 {
	return make(chan int32)
}


func dequeueHandler(server chan Message, client chan int32) {
	if v, err := Dequeue(server, client); err != nil {
		fmt.Println("Error: ", err)
	} else {
		fmt.Println("Dequeue: ", v)
	}
	time.Sleep(500 * time.Millisecond)
}

func main() {
	setting := &ServerSetting{ QueueSize: 10 }
	server := setting.run()

	client1 := CreateClient()
	client2 := CreateClient()

	Enqueue(server, 1, client1)
	Enqueue(server, 2, client2)
	Enqueue(server, 3, client1)
	Enqueue(server, 4, client2)

	dequeueHandler(server, client1)
	dequeueHandler(server, client2)
	dequeueHandler(server, client1)
	dequeueHandler(server, client2)

	setting.down(server)

	time.Sleep(1 * time.Second)
	fmt.Println("all finished!")
}