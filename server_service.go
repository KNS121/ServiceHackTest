package main

import (
    "golang.org/x/sys/windows/svc"
    "golang.org/x/sys/windows/svc/debug"
    "log"
    "net"
    "time"
)

type ServerServiceHackTest struct {
    stopChan chan struct{}
}

func (m *ServerServiceHackTest) Execute(args []string, r <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
    const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
    tick := time.Tick(5 * time.Second)
    m.stopChan = make(chan struct{})

    status <- svc.Status{State: svc.StartPending}
    status <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

    // Запуск TCP-сервера
    go func() {
        listener, err := net.Listen("tcp", ":4545")
        if err != nil {
            log.Printf("Error starting server: %v", err)
            return
        }
        defer listener.Close()
        log.Println("Server is listening...")
        for {
            conn, err := listener.Accept()
            if err != nil {
                log.Printf("Error accepting connection: %v", err)
                return
            }
            go m.handleConnection(conn)
        }
    }()

loop:
    for {
        select {
        case <-tick:
            log.Print("Tick Handled...")
        case c := <-r:
            switch c.Cmd {
            case svc.Interrogate:
                status <- c.CurrentStatus
            case svc.Stop, svc.Shutdown:
                log.Print("Shutting service...!")
                close(m.stopChan)
                break loop
            case svc.Pause:
                status <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
            case svc.Continue:
                status <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
            default:
                log.Printf("Unexpected service control request #%d", c)
            }
        }
    }

    status <- svc.Status{State: svc.StopPending}
    return false, 1
}

func (m *ServerServiceHackTest) handleConnection(conn net.Conn) {
    defer conn.Close()
    message := "Hello, I am a server"
    conn.Write([]byte(message))
    // Добавьте логику для обработки данных от клиента, если необходимо
    buf := make([]byte, 1024)
    for {
        select {
        case <-m.stopChan:
            log.Println("Closing connection due to stop command")
            return
        default:
            n, err := conn.Read(buf)
            if err != nil {
                log.Printf("Error reading from connection: %v", err)
                return
            }
            log.Printf("Received data: %s", buf[:n])
        }
    }
}

func runService(name string, isDebug bool) {
    if isDebug {
        err := debug.Run(name, &ServerServiceHackTest{})
        if err != nil {
            log.Fatalln("Error running service in debug mode:", err)
        }
    } else {
        err := svc.Run(name, &ServerServiceHackTest{})
        if err != nil {
            log.Fatalln("Error running service in Service Control mode:", err)
        }
    }
}

func main() {
    serviceName := "ServerServiceHackTest"
    runService(serviceName, false)
}