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
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.
//
// firewall_test.go contains test cases for firewall.go

package firewall

// Some comments on use of mocking framework in helpers_test.go.

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"

	utilexec "github.com/romana/core/agent/exec"
)

// TestCreateChains is checking that CreateChains generates correct OS commands
// for iptables to create firewall chains.
func TestCreateChains(t *testing.T) {
	// CreateChains doesn't care for output
	// we only want to analize which command generated by the function
	mockExec := &utilexec.FakeExecutor{}

	fw := IPtables{
		os:            mockExec,
		Store:         firewallStore{},
		networkConfig: mockNetworkConfig{},
	}
	fw.SetEndpoint(mockFirewallEndpoint{"eth0", "A", net.ParseIP("127.0.0.1")})
	_ = fw.CreateChains(fw.chains)

	cmd1 := fmt.Sprintf("%s -w -L ROMANA-INPUT", iptablesCmd)
	cmd2 := fmt.Sprintf("%s -w -L ROMANA-FORWARD-IN", iptablesCmd)
	cmd3 := fmt.Sprintf("%s -w -L ROMANA-FORWARD-OUT", iptablesCmd)
	cmd4 := fmt.Sprintf("%s -w -L ROMANA-FORWARD-IN", iptablesCmd)
	expect := strings.Join([]string{cmd1, cmd2, cmd3, cmd4}, "\n")

	if *mockExec.Commands != expect {
		t.Errorf("Unexpected input from TestCreateChains, expect\n%s, got\n%s", expect, *mockExec.Commands)
	}
}

// TestDivertTraffic is checking that DivertTrafficToRomanaIPtablesChain generates correct commands for
// firewall to divert traffic into Romana chains.
func TestDivertTraffic(t *testing.T) {
	// We need to simulate failure on response from os.exec
	// so isRuleExist would fail and trigger EnsureRule.
	// But because EnsureRule will use same object as
	// response from os.exec we will see error in test logs,
	// it's ok as long as function generates expected set of commands.
	mockExec := &utilexec.FakeExecutor{Error: errors.New("Rule not found")}

	// Initialize database.
	mockStore := makeMockStore()

	fw := IPtables{
		os:            mockExec,
		Store:         mockStore,
		networkConfig: mockNetworkConfig{},
	}
	fw.SetEndpoint(mockFirewallEndpoint{"eth0", "A", net.ParseIP("127.0.0.1")})
	fw.DivertTrafficToRomanaIPtablesChain(fw.chains[InputChainIndex], installDivertRules)

	cmd1 := fmt.Sprintf("%s -w -C INPUT -i eth0 -j ROMANA-INPUT", iptablesCmd)
	cmd2 := fmt.Sprintf("%s -A INPUT -i eth0 -j ROMANA-INPUT", iptablesCmd)
	expect := strings.Join([]string{cmd1, cmd2}, "\n")

	if *mockExec.Commands != expect {
		t.Errorf("Unexpected input from TestDivertTraffic, expect\n%s, got\n%s", expect, *mockExec.Commands)
	}
	t.Log("All good here, don't be afraid if 'Diverting traffic failed' message")
}

// TestCreateDefaultRules is checking that CreateRules generates correct commands to create
// firewall rules.
func TestCreateDefaultRules(t *testing.T) {
	mockExec := &utilexec.FakeExecutor{}
	ip := net.ParseIP("127.0.0.1")

	// Test default rules wit DROP action
	// Initialize database.
	mockStore := makeMockStore()

	fw := IPtables{
		os:            mockExec,
		Store:         mockStore,
		networkConfig: mockNetworkConfig{},
	}
	fw.SetEndpoint(mockFirewallEndpoint{"eth0", "A", ip})
	fw.CreateDefaultRule(InputChainIndex, targetDrop)

	// expect
	cmd1 := fmt.Sprintf("%s -w -C ROMANA-INPUT -j DROP", iptablesCmd)
	expect := strings.Join([]string{cmd1},
		"\n")

	if *mockExec.Commands != expect {
		t.Errorf("Unexpected input from TestCreateRules, expect\n%s, got\n%s", expect, *mockExec.Commands)
	}

	// Test default rules wit ACCEPT action
	// Re-initialize database.
	mockStore = makeMockStore()

	// Re-initialize exec
	mockExec = &utilexec.FakeExecutor{}

	// Re-initialize IPtables
	fw = IPtables{
		os:            mockExec,
		Store:         mockStore,
		networkConfig: mockNetworkConfig{},
	}
	fw.SetEndpoint(mockFirewallEndpoint{"eth0", "A", ip})
	fw.CreateDefaultRule(InputChainIndex, targetAccept)

	// expect
	cmd1 = fmt.Sprintf("%s -w -C ROMANA-INPUT -j ACCEPT", iptablesCmd)
	expect = strings.Join([]string{cmd1}, "\n")
	if *mockExec.Commands != expect {
		t.Errorf("Unexpected input from TestCreateRules, expect\n%s, got\n%s", expect, *mockExec.Commands)
	}

}

// TestCreateRules is checking that CreateRules generates correct commands to create
// firewall rules.
func TestCreateRules(t *testing.T) {
	// we only care for recorded commands, no need for fake output or errors
	mockExec := &utilexec.FakeExecutor{}

	// Initialize database.
	mockStore := makeMockStore()

	fw := IPtables{
		os:            mockExec,
		Store:         mockStore,
		networkConfig: mockNetworkConfig{},
	}
	fw.SetEndpoint(mockFirewallEndpoint{"eth0", "A", net.ParseIP("127.0.0.1")})

	rule := NewFirewallRule()
	rule.SetBody("ROMANA-INPUT -d 255.255.255.255/32 -p udp -m udp --sport 68 --dport 67 -j ACCEPT")
	rules := []FirewallRule{rule}

	fw.SetDefaultRules(rules)
	err := fw.CreateRules(InputChainIndex)
	if err != nil {
		t.Errorf("Error calling CreateRules - %s", err)
	}

	cmd1 := fmt.Sprintf("%s %s %s",
		iptablesCmd,
		"-w -C ROMANA-INPUT -d 255.255.255.255/32 -p udp",
		"-m udp --sport 68 --dport 67 -j ACCEPT",
	)
	expect := strings.Join([]string{cmd1}, "\n")

	if *mockExec.Commands != expect {
		t.Errorf("Unexpected input from TestCreateRules, expect\n%s, got\n%s", expect, *mockExec.Commands)
	}
}

// TestDisallowEmptySubstringInCleanup checks that calling
// deleteIPtablesRulesBySubstring() with empty argument is an error.
func TestDisallowEmptySubstring(t *testing.T) {
	fw := IPtables{}
	err := fw.deleteIPtablesRulesBySubstring("")
	if err == nil {
		t.Error("Expected an error when calling deleteIPtablesRulesBySubstring(\"\")")
	} else {
		t.Logf("Received %s", err)
	}
}

// TestCreateU32Rule is checking that CreateU32Rules generates correct commands to
// create firewall rules.
func TestCreateU32Rules(t *testing.T) {

	// we only care for recorded commands, no need for fake output or errors
	mockExec := &utilexec.FakeExecutor{}

	// Initialize database.
	mockStore := makeMockStore()

	fw := IPtables{
		os:            mockExec,
		Store:         mockStore,
		networkConfig: mockNetworkConfig{},
	}
	fw.SetEndpoint(mockFirewallEndpoint{"eth0", "A", net.ParseIP("127.0.0.1")})
	fw.CreateU32Rules(InputChainIndex)

	cmd1 := fmt.Sprintf("%s %s %s",
		iptablesCmd,
		"-w -A ROMANA-INPUT -m u32 --u32",
		"12&0xFF00FF00=0x7F000000&&16&0xFF00FF00=0x7F000000 -j ACCEPT",
	)
	expect := strings.Join([]string{cmd1}, "\n")

	if *mockExec.Commands != expect {
		t.Errorf("Unexpected input from TestCreateU32Rules, expect\n%s, got\n%s", expect, *mockExec.Commands)
	}
}