// Copyright (c) 2016 Pani Networks
// All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

// firewall_test.go contains test cases for firewall.go
package agent

// Some comments on use of mocking framework in helpers_test.go.

import (
	"errors"
	"net"
	"strings"
	"testing"
)

// TestNewChains is checking that detectMissingChains correctly detects which
// Romana chains must be created for given NetIf.
func TestNewChains(t *testing.T) {
	agent := mockAgent()

	// detectMissingChains calls isChainExist which is reading FakeExecutor
	// isChainExist doesn't care for output but must receive not nil error
	// otherwise it would decide that chain exist already and skip
	E := &FakeExecutor{nil, errors.New("bla"), nil}
	agent.Helper.Executor = E
	ip := net.ParseIP("127.0.0.1")
	fw, _ := NewFirewall(NetIf{"eth0", "A", ip}, &agent)

	newChains := fw.detectMissingChains()

	if len(newChains) != 3 {
		t.Error("TestNewChains failed")
	}

	// TODO test case with some chains already exist requires support for
	// stack of output in FakeExecutor
}

// TestCreateChains is checking that CreateChains generates correct OS commands
// for iptables to create firewall chains.
func TestCreateChains(t *testing.T) {
	agent := mockAgent()

	// CreateChains doesn't care for output and we don't any errors
	// we only want to analize which command generated by the function
	E := &FakeExecutor{nil, nil, nil}
	agent.Helper.Executor = E
	ip := net.ParseIP("127.0.0.1")
	fw, _ := NewFirewall(NetIf{"eth0", "A", ip}, &agent)

	_ = fw.CreateChains([]int{0, 1, 2})

	expect := strings.Join([]string{"/sbin/iptables -N ROMANA-T0S0-INPUT",
		"/sbin/iptables -N ROMANA-T0S0-OUTPUT",
		"/sbin/iptables -N ROMANA-T0S0-FORWARD"}, "\n")

	if *E.Commands != expect {
		t.Errorf("Unexpected input from TestCreateChains, expect\n%s, got\n%s", expect, *E.Commands)
	}
}

// TestDivertTraffic is checking that DivertTrafficToRomanaIptablesChain generates correct commands for
// firewall to divert traffic into Romana chains.
func TestDivertTraffic(t *testing.T) {
	agent := mockAgent()

	// We need to simulate failure on response from os.exec
	// so isRuleExist would fail and trigger ensureIptablesRule.
	// But because ensureIptablesRule will use same object as
	// response from os.exec we will see error in test logs,
	// it's ok as long as function generates expected set of commands.
	E := &FakeExecutor{nil, errors.New("Rule not found"), nil}
	agent.Helper.Executor = E
	ip := net.ParseIP("127.0.0.1")
	fw, _ := NewFirewall(NetIf{"eth0", "A", ip}, &agent)

	// 0 is a first standard chain - INPUT
	fw.DivertTrafficToRomanaIptablesChain(0)

	expect := "/sbin/iptables -C INPUT -i eth0 -j ROMANA-T0S0-INPUT\n/sbin/iptables -A INPUT -i eth0 -j ROMANA-T0S0-INPUT"

	if *E.Commands != expect {
		t.Errorf("Unexpected input from TestDivertTraffic, expect\n%s, got\n%s", expect, *E.Commands)
	}
	t.Log("All good here, don't be afraid if 'Diverting traffic failed' message")
}

// TestCreateRules is checking that CreateRules generates correct commands to create
// firewall rules.
func TestCreateRules(t *testing.T) {
	agent := mockAgent()

	// we only care for recorded commands, no need for fake output or errors
	E := &FakeExecutor{nil, nil, nil}
	agent.Helper.Executor = E
	ip := net.ParseIP("127.0.0.1")
	fw, _ := NewFirewall(NetIf{"eth0", "A", ip}, &agent)

	// 0 is a first standard chain - INPUT
	fw.CreateRules(0)

	expect := strings.Join([]string{
		"/sbin/iptables -A ROMANA-T0S0-INPUT -d 172.17.0.1/32 -p icmp -m icmp --icmp-type 0 -m state --state RELATED,ESTABLISHED -j ACCEPT",
		"/sbin/iptables -A ROMANA-T0S0-INPUT -d 172.17.0.1/32 -p tcp -m tcp --sport 22 -j ACCEPT",
		"/sbin/iptables -A ROMANA-T0S0-INPUT -d 255.255.255.255/32 -p udp -m udp --sport 68 --dport 67 -j ACCEPT",
	}, "\n")

	if *E.Commands != expect {
		t.Errorf("Unexpected input from TestCreateRules, expect\n%s, got\n%s", expect, *E.Commands)
	}
}

// TestCreateU32Rule is checking that CreateU32Rules generates correct commands to
// create firewall rules.
func TestCreateU32Rules(t *testing.T) {
	agent := mockAgent()

	// we only care for recorded commands, no need for fake output or errors
	E := &FakeExecutor{nil, nil, nil}

	agent.Helper.Executor = E
	ip := net.ParseIP("127.0.0.1")
	fw, _ := NewFirewall(NetIf{"eth0", "A", ip}, &agent)

	// 0 is a first standard chain - INPUT
	fw.CreateU32Rules(0)

	expect := strings.Join([]string{"/sbin/iptables -A ROMANA-T0S0-INPUT -m u32 --u32 12&0xFF00FF00=0x7F000000 && 16&0xFF00FF00=0x7F000000 -j ACCEPT"}, "\n")

	if *E.Commands != expect {
		t.Errorf("Unexpected input from TestCreateU32Rules, expect\n%s, got\n%s", expect, *E.Commands)
	}
}
