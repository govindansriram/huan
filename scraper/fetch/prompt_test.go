package fetch

//import (
//	"fmt"
//	"strconv"
//	"testing"
//	"time"
//)
//
//func workSimulator(str string) (error, string) {
//	fmt.Println(str)
//	time.Sleep(2 * time.Second)
//	return nil, "123"
//}
//
//func Test_promptPool(t *testing.T) {
//
//	stringSlice := make([]*string, 20)
//
//	for i := range 20 {
//		str := strconv.Itoa(i)
//		stringSlice[i] = &str
//	}
//
//	promptPool(5, workSimulator, stringSlice)
//}
