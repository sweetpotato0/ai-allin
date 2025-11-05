package main

import (
	"fmt"

	"github.com/sweetpotato0/ai-allin/context"
	"github.com/sweetpotato0/ai-allin/message"
)

func main() {
	fmt.Println("=== Context Management Example ===")

	// Create a new context with default max size (100 messages)
	ctx := context.New()

	// Add various messages
	fmt.Println("Adding messages to context...")

	// Add system message
	ctx.AddMessage(message.NewMessage(message.RoleSystem, "You are a helpful AI assistant."))

	// Add conversation
	ctx.AddMessage(message.NewMessage(message.RoleUser, "Hello!"))
	ctx.AddMessage(message.NewMessage(message.RoleAssistant, "Hi! How can I help you today?"))
	ctx.AddMessage(message.NewMessage(message.RoleUser, "What's the weather like?"))
	ctx.AddMessage(message.NewMessage(message.RoleAssistant, "I don't have access to real-time weather data."))

	// Display all messages
	fmt.Printf("\nCurrent context size: %d messages\n", ctx.Size())
	fmt.Println("\nAll messages:")
	for i, msg := range ctx.GetMessages() {
		fmt.Printf("%d. [%s] %s\n", i+1, msg.Role, msg.Content)
	}

	// Get messages by role
	fmt.Println("\nUser messages only:")
	userMsgs := ctx.GetMessagesByRole(message.RoleUser)
	for i, msg := range userMsgs {
		fmt.Printf("%d. %s\n", i+1, msg.Content)
	}

	fmt.Println("\nAssistant messages only:")
	assistantMsgs := ctx.GetMessagesByRole(message.RoleAssistant)
	for i, msg := range assistantMsgs {
		fmt.Printf("%d. %s\n", i+1, msg.Content)
	}

	// Get last message
	lastMsg := ctx.GetLastMessage()
	fmt.Printf("\nLast message: [%s] %s\n", lastMsg.Role, lastMsg.Content)

	// Demonstrate context size management
	fmt.Println("\n=== Testing Context Size Limit ===")

	// Create context with small max size
	smallCtx := context.NewWithMaxSize(5)

	// Add system message (will be preserved)
	smallCtx.AddMessage(message.NewMessage(message.RoleSystem, "System instruction"))

	// Add many messages to exceed limit
	for i := 1; i <= 10; i++ {
		smallCtx.AddMessage(message.NewMessage(message.RoleUser, fmt.Sprintf("Message %d", i)))
		smallCtx.AddMessage(message.NewMessage(message.RoleAssistant, fmt.Sprintf("Response %d", i)))
	}

	fmt.Printf("After adding 20 messages (with max size 5):\n")
	fmt.Printf("Actual size: %d messages\n", smallCtx.Size())
	fmt.Println("\nRemaining messages (system + recent):")
	for i, msg := range smallCtx.GetMessages() {
		fmt.Printf("%d. [%s] %s\n", i+1, msg.Role, msg.Content)
	}

	// Clear context
	fmt.Println("\n=== Clearing Context ===")
	ctx.Clear()
	fmt.Printf("Context size after clear: %d\n", ctx.Size())

	// Practical use case: Simulating a conversation with context awareness
	fmt.Println("\n=== Practical Conversation Example ===")
	conversationCtx := context.NewWithMaxSize(10)

	// System prompt
	conversationCtx.AddMessage(message.NewMessage(message.RoleSystem, "You are a helpful shopping assistant."))

	// Multi-turn conversation
	conversations := []struct {
		user      string
		assistant string
	}{
		{"I'm looking for a laptop", "Great! What's your budget and primary use case?"},
		{"Under $1000, for programming", "I recommend looking at laptops with at least 16GB RAM and SSD storage."},
		{"What brands do you suggest?", "Popular choices for developers include Dell XPS, Lenovo ThinkPad, and MacBook Air."},
		{"Tell me more about the Dell XPS", "The Dell XPS series offers excellent build quality, good battery life, and powerful processors."},
	}

	for _, conv := range conversations {
		conversationCtx.AddMessage(message.NewMessage(message.RoleUser, conv.user))
		conversationCtx.AddMessage(message.NewMessage(message.RoleAssistant, conv.assistant))
	}

	fmt.Println("Conversation history:")
	for _, msg := range conversationCtx.GetMessages() {
		if msg.Role == message.RoleSystem {
			continue
		}
		fmt.Printf("[%s] %s\n", msg.Role, msg.Content)
	}

	fmt.Printf("\nTotal messages in context: %d\n", conversationCtx.Size())
	fmt.Println("\n=== Context Features ===")
	fmt.Println("✓ Automatic message history management")
	fmt.Println("✓ Configurable size limits")
	fmt.Println("✓ System message preservation")
	fmt.Println("✓ Filter messages by role")
	fmt.Println("✓ Thread-safe operations")
}
