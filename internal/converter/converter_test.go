package converter

import (
	"bytes"
	"encoding/json"
	"testing"

	xserial "github.com/xtls/xray-core/infra/conf/serial"

	_ "github.com/xtls/xray-core/main/distro/all"
)

// v2go renames every config before it is tested, so every link reaching
// ConvertLink carries a fragment like "#v2go | 🇩🇪 DE | VLESS | 12" —
// spaces, pipes and emoji included. These cases pin that down: the link must
// convert, and the resulting outbound must be something xray will actually load.
func TestConvertLinkWithV2goFragment(t *testing.T) {
	cases := []struct {
		name string
		link string
	}{
		{
			"vless-reality",
			"vless://1cbe9e8a-8e2f-4a1e-9a2f-0e6f9a3b7c11@example.com:443" +
				"?type=tcp&security=reality&sni=www.microsoft.com&fp=chrome" +
				"&pbk=xNfLwSFtT5ZK8Q3iZ6ub1t7z0oUqNVoJ0aQpM2vFhHU&sid=6ba85179e30d4fc2" +
				"&flow=xtls-rprx-vision#v2go | 🇩🇪 DE | VLESS | 12",
		},
		{
			"vless-ws-tls",
			"vless://1cbe9e8a-8e2f-4a1e-9a2f-0e6f9a3b7c11@example.com:443" +
				"?type=ws&security=tls&path=%2Fws&host=example.com&sni=example.com" +
				"#v2go | 🇳🇱 NL | VLESS | 3",
		},
		{
			// base64 userinfo, the shape that actually shows up in the wild
			"ss-base64-userinfo",
			"ss://YWVzLTI1Ni1nY206cGFzc3dvcmQ=@example.com:8388#v2go | 🇺🇸 US | SS | 7",
		},
		{
			"trojan-tls",
			"trojan://somepassword@example.com:443?security=tls&sni=example.com&type=tcp" +
				"#v2go | 🇫🇷 FR | TROJAN | 1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			outbound, err := ConvertLink(tc.link)
			if err != nil {
				t.Fatalf("ConvertLink: %v", err)
			}

			full := M{
				"log": M{"loglevel": "none"},
				"outbounds": []M{
					outbound,
					{"protocol": "freedom", "tag": "direct"},
				},
			}
			raw, err := json.Marshal(full)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if _, err := xserial.LoadJSONConfig(bytes.NewReader(raw)); err != nil {
				t.Fatalf("xray rejected the generated config: %v\n%s", err, raw)
			}
		})
	}
}

func TestConvertLinkRejectsGarbage(t *testing.T) {
	for _, link := range []string{"", "not-a-link", "vless://", "ss://@@@"} {
		if _, err := ConvertLink(link); err == nil {
			t.Errorf("expected error for %q", link)
		}
	}
}
