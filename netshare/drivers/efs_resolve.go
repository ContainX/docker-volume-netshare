package drivers

import (
	"fmt"
	"github.com/miekg/dns"
)

var (
	ErrorEmpty = fmt.Errorf("Response was empty")
	ErrorParse = fmt.Errorf("Could not parse A record")
)

type Resolver struct {
	serverString string
}

type Lookup interface {
	Lookup(name string) (string, error)
}

func NewDefaultResolver() *Resolver {
	config, _ := dns.ClientConfigFromFile("/etc/resolv.conf")
	serverString := config.Servers[0] + ":" + config.Port
	l := new(Resolver)
	l.serverString = serverString
	return l
}
func NewResolver(serverString string) *Resolver {
	if serverString == "" {
		return NewDefaultResolver()
	}
	l := new(Resolver)
	l.serverString = serverString + ":53"
	return l
}

func (l *Resolver) Lookup(name string) (string, error) {
	answer, err := l.lookup(name, "udp")
	if err != nil {
		return "", err
	}
	return l.parseAnswer(answer)
}

func (l *Resolver) lookup(name string, connType string) (*dns.Msg, error) {
	qType := dns.TypeA
	name = dns.Fqdn(name)

	client := &dns.Client{Net: connType}

	msg := &dns.Msg{}
	msg.SetQuestion(name, qType)

	response, _, err := client.Exchange(msg, l.serverString)

	if err != nil {
		if connType == "" {
			return l.lookup(name, "tcp")
		} else {
			return nil, fmt.Errorf("Couldn't resolve name '%s' : %s", name, err.Error())
		}
	}

	if msg.Id != response.Id {
		return nil, fmt.Errorf("DNS ID mismatch, request: %d, response: %d", msg.Id, response.Id)
	}
	return response, nil
}

func (l *Resolver) parseAnswer(answer *dns.Msg) (string, error) {
	if len(answer.Answer) == 0 {
		return "", ErrorEmpty
	}
	if a, ok := answer.Answer[0].(*dns.A); ok {
		return a.A.String(), nil
	}
	return "", ErrorParse
}
