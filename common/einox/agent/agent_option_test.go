package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptions_SkillsDir(t *testing.T) {
	opts := &options{
		skillsDir: "/path/to/skills",
	}
	assert.Equal(t, "/path/to/skills", opts.skillsDir)
}

func TestOptions_DefaultValues(t *testing.T) {
	opts := &options{}
	assert.Equal(t, "", opts.name)
	assert.Equal(t, "", opts.skillsDir)
	assert.False(t, opts.stream)
	assert.False(t, opts.enableWriteTodos)
	assert.False(t, opts.enableFileSystem)
}
