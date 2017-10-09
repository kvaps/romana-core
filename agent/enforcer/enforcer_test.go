package enforcer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/romana/core/agent/internal/cache/policycache"
	"github.com/romana/core/agent/iptsave"
	"github.com/romana/core/common/api"
	"github.com/romana/core/pkg/policytools"
	"github.com/romana/ipset"
)

func TestMakePolicyRules(t *testing.T) {
	makeEmptyIptables := func() iptsave.IPtables {
		return iptsave.IPtables{
			Tables: []*iptsave.IPtable{
				&iptsave.IPtable{
					Name: "filter",
				},
			},
		}
	}

	makeEndpoints := func(endpoints ...api.Endpoint) (result []api.Endpoint) {
		for _, e := range endpoints {
			result = append(result, e)
		}
		return
	}

	withCidr := func(s ...string) api.Endpoint {
		return api.Endpoint{Cidr: s[0]}
	}
	withTenant := func(t ...string) api.Endpoint {
		return api.Endpoint{TenantID: t[0]}
	}
	withTenantSegment := func(s ...string) api.Endpoint {
		return api.Endpoint{TenantID: s[0], SegmentID: s[1]}
	}
	_ = withTenantSegment

	makeRules := func(rules ...api.Rule) (result []api.Rule) {
		for _, r := range rules {
			result = append(result, r)
		}
		return result
	}
	withProtoPorts := func(proto string, ports ...uint) api.Rule {
		return api.Rule{Protocol: proto, Ports: ports}
	}

	/*
		blocks := []api.IPAMBlockResponse{
			api.IPAMBlockResponse{
				Tenant:  "T800",
				Segment: "John",
			},
			api.IPAMBlockResponse{
				Tenant:  "T1000",
				Segment: "",
			},
			api.IPAMBlockResponse{
				Tenant:  "T3000",
				Segment: "",
			},
			api.IPAMBlockResponse{
				Tenant:  "T100K",
				Segment: "skynet",
			},
		}
	*/

	testCases := []struct {
		name   string
		schema string
		policy api.Policy
	}{
		{
			name:   "ingress basic",
			schema: policytools.SchemePolicyOnTop,
			policy: api.Policy{
				ID:        "<TESTPOLICYID>",
				Direction: api.PolicyDirectionIngress,
				AppliedTo: makeEndpoints(withTenant("T1000")),
				Ingress: []api.RomanaIngress{
					api.RomanaIngress{
						Peers: makeEndpoints(withCidr("10.0.0.0/99")),
						Rules: makeRules(withProtoPorts("TCP", 80, 99, 8080)),
					},
				},
			},
		},
		{
			name:   "egress basic",
			schema: policytools.SchemeTargetOnTop,
			policy: api.Policy{
				ID:        "<TESTPOLICYID>",
				Direction: api.PolicyDirectionEgress,
				AppliedTo: makeEndpoints(withTenant("T1000"), withTenantSegment("T800", "John")),
				Ingress: []api.RomanaIngress{
					api.RomanaIngress{
						Peers: makeEndpoints(
							withCidr("10.0.0.0/99"),
							withTenant("T3000"),
							withTenantSegment("T100K", "skynet")),
						Rules: makeRules(
							withProtoPorts("TCP", 80, 99, 8080),
							withProtoPorts("UDP", 53, 1194),
						),
					},
				},
			},
		},
	}

	toList := func(p ...api.Policy) []api.Policy {
		return p
	}

	noop := func(target api.Endpoint) bool { return true }

	for _, tc := range testCases {
		sets := ipset.Ipset{}
		iptables := makeEmptyIptables()
		makePolicies(toList(tc.policy), noop, &iptables)
		t.Log(iptables.Render())
		t.Log(sets.Render(ipset.RenderCreate))
	}
}

func TestMakePolicySets(t *testing.T) {
	makeEndpoints := func(endpoints ...api.Endpoint) (result []api.Endpoint) {
		for _, e := range endpoints {
			result = append(result, e)
		}
		return
	}

	withCidr := func(s ...string) api.Endpoint {
		return api.Endpoint{Cidr: s[0]}
	}
	withTenant := func(t ...string) api.Endpoint {
		return api.Endpoint{TenantID: t[0]}
	}
	withTenantSegment := func(s ...string) api.Endpoint {
		return api.Endpoint{TenantID: s[0], SegmentID: s[1]}
	}
	/*
		makeRules := func(rules ...api.Rule) (result []api.Rule) {
			for _, r := range rules {
				result = append(result, r)
			}
			return result
		}
		withProtoPorts := func(proto string, ports ...uint) api.Rule {
			return api.Rule{Protocol: proto, Ports: ports}
		}
	*/

	// expectFunc is a signature for a function used in test cases to
	// assert test success.
	type expectFunc func(*ipset.Set, error) error

	// return expectFunc that looks for provided cidrs in Set.
	matchIpsetMember := func(cidrs ...string) expectFunc {
		return func(set *ipset.Set, err error) error {
			for _, cidr := range cidrs {
				for _, member := range set.Members {
					if member.Elem == cidr {
						// found
						continue
					}

					return fmt.Errorf("cidr %s not found in set %s",
						cidr, set)
				}
			}

			return nil
		}

	}

	testCases := []struct {
		name   string
		policy api.Policy
		expect expectFunc
	}{
		{
			name: "ingress sets basic",
			policy: api.Policy{
				ID:        "<TESTPOLICYID>",
				Direction: api.PolicyDirectionIngress,
				AppliedTo: makeEndpoints(withTenant("T1000")),
				Ingress: []api.RomanaIngress{
					api.RomanaIngress{
						Peers: makeEndpoints(withCidr("10.0.0.0/99")),
					},
				},
			},
			expect: matchIpsetMember("10.0.0.0/99"),
		},
		{
			name: "egress sets basic",
			policy: api.Policy{
				ID:        "<TESTPOLICYID>",
				Direction: api.PolicyDirectionEgress,
				AppliedTo: makeEndpoints(withTenant("T1000"), withTenantSegment("T800", "John")),
				Ingress: []api.RomanaIngress{
					api.RomanaIngress{
						Peers: makeEndpoints(
							withCidr("10.0.0.0/99"),
							withTenant("T3000"),
							withTenantSegment("T100K", "skynet")),
					},
				},
			},
			expect: matchIpsetMember("10.0.0.0/99"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			set1, err := makePolicySets(tc.policy)
			sets := ipset.Ipset{Sets: []*ipset.Set{set1}}
			t.Log(sets.Render(ipset.RenderSave))

			err = tc.expect(set1, err)
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestMakeBlockSets(t *testing.T) {

	makeCIDR := func(s string) api.IPNet {
		_, ipnet, _ := net.ParseCIDR(s)
		return api.IPNet{*ipnet}
	}

	testCases := []struct {
		name       string
		hostname   string
		blockCache []api.IPAMBlockResponse
	}{
		{
			name:     "basic 1",
			hostname: "host1",
			blockCache: []api.IPAMBlockResponse{
				api.IPAMBlockResponse{
					Tenant:  "T800",
					Segment: "john",
					CIDR:    makeCIDR("10.0.0.0/28"),
				},
				api.IPAMBlockResponse{
					Tenant:  "T100k",
					Segment: "skynet",
					CIDR:    makeCIDR("10.1.0.0/28"),
				},
			},
		},
	}

	for _, tc := range testCases {
		sets, err := makeBlockSets(tc.blockCache, policycache.New(), tc.hostname)
		t.Log(sets.Render(ipset.RenderSave))
		t.Log(err)
	}
}

var tdir = "testdata"

func TestMakePolicies(t *testing.T) {
	files, err := ioutil.ReadDir(tdir)
	if err != nil {
		t.Skip("Folder with test data not found")
	}
	_ = files

	loadRomanaPolicy := func(file string) (*api.Policy, error) {
		data, err := ioutil.ReadFile(filepath.Join(tdir, file))
		if err != nil {
			return nil, err
		}

		var policy api.Policy

		err = json.Unmarshal(data, &policy)

		if err != nil {
			return nil, err
		}

		return &policy, nil
	}

	toList := func(p ...api.Policy) []api.Policy {
		return p
	}

	noop := func(target api.Endpoint) bool { return true }
	_ = loadRomanaPolicy

	test := func(file string, t *testing.T) func(*testing.T) {
		return func(t *testing.T) {
			policy, err := loadRomanaPolicy(file)
			if err != nil {
				t.Fatal(err)
			}

			iptables := iptsave.IPtables{
				Tables: []*iptsave.IPtable{
					&iptsave.IPtable{
						Name: "filter",
					},
				},
			}

			makePolicies(toList(*policy), noop, &iptables)

			referenceName := strings.Replace(file, ".json", ".iptables", -1)

			// generate golden files
			if os.Getenv("MAKE_GOLD") != "" {
				err = ioutil.WriteFile(filepath.Join(tdir, referenceName), []byte(iptables.Render()), 0644)
				if err != nil {
					t.Fatal(err)
				}

				return
			}

			referenceFile, err := ioutil.ReadFile(filepath.Join(tdir, referenceName))
			if err != nil {
				t.Fatal(err)
			}

			if string(referenceFile) != iptables.Render() {
				t.Fatal(file)
			}
		}
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") {
			t.Run(file.Name(), test(file.Name(), t))
		}
	}
}
