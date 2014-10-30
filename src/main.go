package main

import (
    "github.com/frankbb/xmpp"
    "encoding/xml"
    "bytes"
    "fmt"
    "log"
    "os"
)

type ConfigContacts struct {
    Contacts []ConfigContact `xml:"contacts>contact"`
}

type ConfigContact struct {
    Name  string `xml:"name"`
    Email string `xml:"email"`
}

func main() {
    domain := "__XMPPDOMAIN__"
    host, port, err := xmpp.Resolve(domain)
    if err != nil {
        log.Fatal("Failed to resolve XMPP server: ", err.Error())
    }

    var addr string
    var user string
    var password string

    xmppConfig := &xmpp.Config{
        Create:         false,
        TrustedAddress: true,
        Archive:        false,
        Resource:       "__XMPPRESOURCE__",
        Log:            os.Stdout,
        InLog:          os.Stdout,
        OutLog:         os.Stdout,
    }

    addr = fmt.Sprintf("%s:%d", host, port)
    user = "__XMPPUSER__"
    password = "__XMPPPASSWORD__"

    conn, err := xmpp.Dial(addr, user, domain, password, xmppConfig)
    if err != nil {
        log.Fatal("Failed to connect to XMPP server: ", err.Error())
    }

    stanzaChan := make(chan xmpp.Stanza)

    go waitForXMPPMessages(conn, stanzaChan)

    for {
        select {
        case rawStanza, ok := <-stanzaChan:
            if !ok {
                continue
            }
            switch stanza := rawStanza.Value.(type) {
            case *xmpp.ClientIQ:
                if stanza.Type != "get" && stanza.Type != "set" {
                    continue
                }
                reply := processIQ(stanza)
                if reply == nil {
                    reply = xmpp.ErrorReply{
                        Type:  "cancel",
                        Error: xmpp.ErrorBadRequest{},
                    }
                }
                if err := conn.SendIQReply(stanza.From, "result", stanza.Id, reply); err != nil {
                    log.Print("Failed to send IQ reply: ", err.Error())
                    continue
                }
            default:
                continue
            }
        }
    }
}

func waitForXMPPMessages(conn *xmpp.Conn, stanzaChan chan<- xmpp.Stanza) {
    defer close(stanzaChan)

    for {
        stanza, err := conn.Next()
        if err != nil {
            log.Print("Next Stanza read failed: ", err.Error())
            return
        }
        stanzaChan <- stanza
    }
}

func processIQ(stanza *xmpp.ClientIQ) interface{} {
    buf := bytes.NewBuffer(stanza.Query)
    parser := xml.NewDecoder(buf)
    token, _ := parser.Token()
    if token == nil {
        return nil
    }
    startElem, ok := token.(xml.StartElement)
    if !ok {
        return nil
    }
    switch startElem.Name.Local {
    case "update_contacts":
        var contacts ConfigContacts
        if err := xml.NewDecoder(bytes.NewBuffer(stanza.Query)).Decode(&contacts); err != nil {
            log.Print("Failed to parse IQ for ConfigContacts")
            return nil
        }
        for _,contact := range contacts.Contacts {
            handleContact(contact)
        }
        return xmpp.EmptyReply{}
    default:
        log.Print("Unknown IQ: ", startElem.Name.Local)
    }

    return nil
}

func handleContact(contact ConfigContact) {
    log.Print("Email: ", contact.Email)
    log.Print("DisplayName: ", contact.Name)
    log.Print("------------------")
}
