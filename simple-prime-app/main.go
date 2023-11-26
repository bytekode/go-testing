package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func main() {
	// n := 7
	// _, msg := isPrime(n)
	// fmt.Println(msg)

	// print a welcome message
	intro()

	// create a channel to indicate when a user wants to quit
	doneChan := make(chan bool)

	// start a goroutine to read user input and run progra
	go readUserInput(doneChan, os.Stdout)

	// block until doneChan gets a value
	<-doneChan

	// close the channel
	close(doneChan)

	// say good bye
	fmt.Println("Goodbye!")

}

func readUserInput(doneChan chan bool, in io.Reader) {
	scanner := bufio.NewScanner(in)
	for {
		res, done := checkNumbers(scanner)
		if done {
			doneChan <- true
			return
		}
		fmt.Println(res)
		prompt()
	}
}

func checkNumbers(scanner *bufio.Scanner) (string, bool) {
	scanner.Scan()

	// check to see if user wants to quit
	if strings.EqualFold(scanner.Text(), "q") {
		return "", true
	}

	// convert user input to int

	numToCheck, err := strconv.Atoi(scanner.Text())
	if err != nil {
		return "Please enter a whole number!", false
	}

	_, msg := isPrime(numToCheck)
	return msg, false
}

func intro() {
	fmt.Println("Is it Prime?")
	fmt.Println("------------")
	fmt.Println("Enter a whole number, and we'll tell you if it is a prime number or not. Enter q to quit.")
	prompt()
}

func prompt() {
	fmt.Print("=> ")
}

func isPrime(n int) (bool, string) {
	// 0 & 1 are not prime by definition
	if n == 0 || n == 1 {
		return false, fmt.Sprintf("%d is not prime, by definition!", n)
	}

	if n < 0 {
		return false, "Negative numbers are not prime, by definition!"
	}

	// use the modulous operator repeadly

	for i := 2; i <= n/2; i++ {
		if n%i == 0 {
			return false, fmt.Sprintf("%d is not prime because it is divisible by %d!", n, i)
		}
	}

	return true, fmt.Sprintf("%d is a prime number!", n)
}
