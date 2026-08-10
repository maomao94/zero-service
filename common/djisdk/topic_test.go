package djisdk

import "testing"

func TestTopicConstructors(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "osd", got: OsdTopic("dock-1"), want: "thing/product/dock-1/osd"},
		{name: "osd_pattern", got: OsdTopicPattern(), want: "thing/product/+/osd"},
		{name: "state", got: StateTopic("dock-1"), want: "thing/product/dock-1/state"},
		{name: "state_pattern", got: StateTopicPattern(), want: "thing/product/+/state"},
		{name: "services", got: ServicesTopic("dock-1"), want: "thing/product/dock-1/services"},
		{name: "services_reply_pattern", got: servicesReplyTopicPattern(), want: "thing/product/+/services_reply"},
		{name: "events", got: EventsTopic("dock-1"), want: "thing/product/dock-1/events"},
		{name: "events_pattern", got: EventsTopicPattern(), want: "thing/product/+/events"},
		{name: "events_reply", got: EventsReplyTopic("dock-1"), want: "thing/product/dock-1/events_reply"},
		{name: "requests_pattern", got: RequestsTopicPattern(), want: "thing/product/+/requests"},
		{name: "requests_reply", got: RequestsReplyTopic("dock-1"), want: "thing/product/dock-1/requests_reply"},
		{name: "property_set", got: PropertySetTopic("dock-1"), want: "thing/product/dock-1/property/set"},
		{name: "property_set_reply_pattern", got: propertySetReplyTopicPattern(), want: "thing/product/+/property/set_reply"},
		{name: "status", got: StatusTopic("dock-1"), want: "sys/product/dock-1/status"},
		{name: "status_pattern", got: StatusTopicPattern(), want: "sys/product/+/status"},
		{name: "status_reply", got: StatusReplyTopic("dock-1"), want: "sys/product/dock-1/status_reply"},
		{name: "drc_up", got: DrcUpTopic("dock-1"), want: "thing/product/dock-1/drc/up"},
		{name: "drc_up_pattern", got: DrcUpTopicPattern(), want: "thing/product/+/drc/up"},
		{name: "drc_down", got: DrcDownTopic("dock-1"), want: "thing/product/dock-1/drc/down"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestTopicPatternsMatchConcreteTopics(t *testing.T) {
	patterns := []string{
		OsdTopicPattern(), StateTopicPattern(), servicesReplyTopicPattern(),
		EventsTopicPattern(), RequestsTopicPattern(), propertySetReplyTopicPattern(),
		StatusTopicPattern(), DrcUpTopicPattern(),
	}
	concrete := []string{
		OsdTopic("a"), StateTopic("a"), ServicesTopic("a") + "_reply",
		EventsTopic("a"), RequestsReplyTopic("a"), PropertySetTopic("a") + "_reply",
		StatusTopic("a"), DrcUpTopic("a"),
	}
	for i := range patterns {
		if patterns[i] == concrete[i] {
			t.Fatalf("pattern %q must not equal concrete topic %q", patterns[i], concrete[i])
		}
	}
}
