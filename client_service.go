package main
import (
    "fmt"
    "os"
    "net"
    "io"
	"bufio"
)
func main() {
 
    conn, err := net.Dial("tcp", "127.0.0.1:4545") 
    if err != nil { 
        fmt.Println(err) 
        return
    } 
    defer conn.Close() 
  
    io.Copy(os.Stdout, conn) 
    fmt.Println("\nDone")

	fmt.Println("Press Enter to exit...")
    reader := bufio.NewReader(os.Stdin)
    reader.ReadString('\n')
}