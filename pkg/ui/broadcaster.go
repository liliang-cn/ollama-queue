
package ui

import (
	"log"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/liliang-cn/ollama-queue/pkg/models"
	"github.com/liliang-cn/ollama-queue/pkg/queue"
)

// Broadcaster broadcasts queue updates to WebSocket clients.

type Broadcaster struct {
	manager    *queue.QueueManager
	clients    map[*websocket.Conn]bool
	clientsMu  sync.RWMutex
	eventChan  <-chan *models.TaskEvent
	stopChan   chan struct{}
}

// NewBroadcaster creates a new Broadcaster.
func NewBroadcaster(manager *queue.QueueManager) (*Broadcaster, error) {
	eventChan, err := manager.Subscribe([]string{"task_submitted", "task_started", "task_completed", "task_cancelled", "priority_updated"})
	if err != nil {
		return nil, err
	}

	b := &Broadcaster{
		manager:   manager,
		clients:   make(map[*websocket.Conn]bool),
		stopChan:  make(chan struct{}),
		eventChan: eventChan,
	}

	go b.run()

	return b, nil
}

// AddClient adds a new WebSocket client.
func (b *Broadcaster) AddClient(conn *websocket.Conn) {
	b.clientsMu.Lock()
	b.clients[conn] = true
	b.clientsMu.Unlock()

	// Send initial task list
	b.broadcastTasks()
}

// RemoveClient removes a WebSocket client.
func (b *Broadcaster) RemoveClient(conn *websocket.Conn) {
	b.clientsMu.Lock()
	delete(b.clients, conn)
	b.clientsMu.Unlock()
}

// Stop stops the broadcaster.
func (b *Broadcaster) Stop() {
	close(b.stopChan)
	b.manager.Unsubscribe(b.eventChan)
}

func (b *Broadcaster) run() {
	for {
		select {
		case <-b.stopChan:
			return
		case <-b.eventChan:
			b.broadcastTasks()
		}
	}
}

func (b *Broadcaster) broadcastTasks() {
	tasks, err := b.manager.ListTasks(models.TaskFilter{})
	if err != nil {
		log.Println("Error getting tasks for broadcast:", err)
		return
	}

	b.clientsMu.RLock()
	defer b.clientsMu.RUnlock()

	for client := range b.clients {
		if err := client.WriteJSON(tasks); err != nil {
			log.Println("Error broadcasting tasks:", err)
			b.RemoveClient(client)
		}
	}
}
