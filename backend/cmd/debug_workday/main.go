package main

import (
	"fmt"
	"name/internal/domain/bazi"
)

func main() {
	hs := bazi.GetHolidayService()
	
	// Test 2024-01-09 (Tuesday)
	isWork := hs.IsWorkDay(2024, 1, 9)
	fmt.Printf("2024-01-09 IsWorkDay: %v\n", isWork)
	
	isVacation := hs.IsVacationDay(2024, 1, 9)
	fmt.Printf("2024-01-09 IsVacationDay: %v\n", isVacation)
	
	// Test 2024-01-08 (Monday)
	isWork2 := hs.IsWorkDay(2024, 1, 8)
	fmt.Printf("2024-01-08 IsWorkDay: %v\n", isWork2)
	
	isVacation2 := hs.IsVacationDay(2024, 1, 8)
	fmt.Printf("2024-01-08 IsVacationDay: %v\n", isVacation2)
	
	// Test 2024-01-06 (Saturday)
	isWork3 := hs.IsWorkDay(2024, 1, 6)
	fmt.Printf("2024-01-06 IsWorkDay: %v\n", isWork3)
	
	isVacation3 := hs.IsVacationDay(2024, 1, 6)
	fmt.Printf("2024-01-06 IsVacationDay: %v\n", isVacation3)
}