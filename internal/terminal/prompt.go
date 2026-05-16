package terminal

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Prompt displays a prompt and waits for user input.
func Prompt(message string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(message + ": ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

// PromptDefault displays a prompt with a default value.
func PromptDefault(message, defaultValue string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	prompt := message
	if defaultValue != "" {
		prompt = fmt.Sprintf("%s [%s]", message, defaultValue)
	}
	prompt += ": "
	fmt.Print(prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue, nil
	}
	return input, nil
}

// Confirm displays a yes/no prompt and returns true for yes.
func Confirm(message string) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(message + " [y/n]: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}
		input = strings.TrimSpace(strings.ToLower(input))
		switch input {
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			fmt.Println("Please enter 'y' or 'n'")
		}
	}
}

// Select displays a selection menu and returns the selected index.
func Select(message string, options []string) (int, error) {
	if len(options) == 0 {
		return -1, fmt.Errorf("no options provided")
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Println(message)
	for i, opt := range options {
		fmt.Printf("  %d) %s\n", i+1, opt)
	}

	for {
		fmt.Print("Select an option: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return -1, err
		}
		input = strings.TrimSpace(input)

		var idx int
		if _, err := fmt.Sscanf(input, "%d", &idx); err != nil {
			fmt.Println("Please enter a number")
			continue
		}

		if idx < 1 || idx > len(options) {
			fmt.Printf("Please enter a number between 1 and %d\n", len(options))
			continue
		}

		return idx - 1, nil
	}
}

// SelectMulti displays a multi-selection menu and returns selected indices.
func SelectMulti(message string, options []string) ([]int, error) {
	if len(options) == 0 {
		return nil, fmt.Errorf("no options provided")
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Println(message)
	for i, opt := range options {
		fmt.Printf("  %d) %s\n", i+1, opt)
	}

	fmt.Println("\nEnter numbers separated by commas (e.g., 1,3,5), or 'all' for all options:")

	for {
		fmt.Print("Select options: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		input = strings.TrimSpace(input)

		if strings.ToLower(input) == "all" {
			result := make([]int, len(options))
			for i := range options {
				result[i] = i
			}
			return result, nil
		}

		parts := strings.Split(input, ",")
		result := make([]int, 0, len(parts))
		seen := make(map[int]bool)

		for _, part := range parts {
			part = strings.TrimSpace(part)
			var idx int
			if _, err := fmt.Sscanf(part, "%d", &idx); err != nil {
				fmt.Println("Invalid input. Enter numbers separated by commas or 'all'")
				continue
			}
			if idx < 1 || idx > len(options) {
				fmt.Printf("Number %d is out of range (1-%d)\n", idx, len(options))
				continue
			}
			if !seen[idx-1] {
				result = append(result, idx-1)
				seen[idx-1] = true
			}
		}

		if len(result) > 0 {
			return result, nil
		}
	}
}
