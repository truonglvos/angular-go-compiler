package shadow_css_test

import (
	"regexp"
	"strings"
	"testing"
)

func TestKeyframes(t *testing.T) {
	t.Run("should scope keyframes rules", func(t *testing.T) {
		css := "@keyframes foo {0% {transform:translate(-50%) scaleX(0);}}"
		expected := "@keyframes host-a_foo {0% {transform:translate(-50%) scaleX(0);}}"
		result := shim(css, "host-a")
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	t.Run("should scope -webkit-keyframes rules", func(t *testing.T) {
		css := "@-webkit-keyframes foo {0% {-webkit-transform:translate(-50%) scaleX(0);}} "
		expected := "@-webkit-keyframes host-a_foo {0% {-webkit-transform:translate(-50%) scaleX(0);}}"
		result := shim(css, "host-a")
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	t.Run("should scope animations using local keyframes identifiers", func(t *testing.T) {
		css := `
        button {
            animation: foo 10s ease;
        }
        @keyframes foo {
            0% {
            transform: translate(-50%) scaleX(0);
            }
        }
        `
		result := shim(css, "host-a")
		if !strings.Contains(result, "animation: host-a_foo 10s ease;") {
			t.Errorf("Expected result to contain 'animation: host-a_foo 10s ease;', got %q", result)
		}
	})

	t.Run("should not scope animations using non-local keyframes identifiers", func(t *testing.T) {
		css := `
        button {
            animation: foo 10s ease;
        }
        `
		result := shim(css, "host-a")
		if !strings.Contains(result, "animation: foo 10s ease;") {
			t.Errorf("Expected result to contain 'animation: foo 10s ease;', got %q", result)
		}
	})

	t.Run("should scope animation-names using local keyframes identifiers", func(t *testing.T) {
		css := `
        button {
            animation-name: foo;
        }
        @keyframes foo {
            0% {
            transform: translate(-50%) scaleX(0);
            }
        }
        `
		result := shim(css, "host-a")
		if !strings.Contains(result, "animation-name: host-a_foo;") {
			t.Errorf("Expected result to contain 'animation-name: host-a_foo;', got %q", result)
		}
	})

	t.Run("should not scope animation-names using non-local keyframes identifiers", func(t *testing.T) {
		css := `
        button {
            animation-name: foo;
        }
        `
		result := shim(css, "host-a")
		if !strings.Contains(result, "animation-name: foo;") {
			t.Errorf("Expected result to contain 'animation-name: foo;', got %q", result)
		}
	})

	t.Run("should handle (scope or not) multiple animation-names", func(t *testing.T) {
		css := `
        button {
            animation-name: foo, bar,baz, qux , quux ,corge ,grault ,garply, waldo;
        }
        @keyframes foo {}
        @keyframes baz {}
        @keyframes quux {}
        @keyframes grault {}
        @keyframes waldo {}`
		result := shim(css, "host-a")
		animationNames := []string{
			"host-a_foo",
			" bar",
			"host-a_baz",
			" qux ",
			" host-a_quux ",
			"corge ",
			"host-a_grault ",
			"garply",
			" host-a_waldo",
		}
		expected := "animation-name: " + strings.Join(animationNames, ",") + ";"
		if !strings.Contains(result, expected) {
			t.Errorf("Expected result to contain %q, got %q", expected, result)
		}
	})

	t.Run("should handle (scope or not) multiple animation-names defined over multiple lines", func(t *testing.T) {
		css := `
        button {
            animation-name: foo,
                            bar,baz,
                            qux ,
                            quux ,
                            grault,
                            garply, waldo;
        }
        @keyframes foo {}
        @keyframes baz {}
        @keyframes quux {}
        @keyframes grault {}`
		result := shim(css, "host-a")
		scoped := []string{"foo", "baz", "quux", "grault"}
		for _, s := range scoped {
			if !strings.Contains(result, "host-a_"+s) {
				t.Errorf("Expected result to contain 'host-a_%s', got %q", s, result)
			}
		}
		nonScoped := []string{"bar", "qux", "garply", "waldo"}
		for _, s := range nonScoped {
			if !strings.Contains(result, s) {
				t.Errorf("Expected result to contain %q, got %q", s, result)
			}
			if strings.Contains(result, "host-a_"+s) {
				t.Errorf("Expected result to not contain 'host-a_%s', got %q", s, result)
			}
		}
	})

	t.Run("should handle (scope or not) animation definition containing some names which do not have a preceding space", func(t *testing.T) {
		COMPONENT_VARIABLE := "%COMP%"
		HOST_ATTR := "_nghost-" + COMPONENT_VARIABLE
		CONTENT_ATTR := "_ngcontent-" + COMPONENT_VARIABLE
		css := `.test {
      animation:my-anim 1s,my-anim2 2s, my-anim3 3s,my-anim4 4s;
    }
    
    @keyframes my-anim {
      0% {color: red}
      100% {color: blue}
    }
    
    @keyframes my-anim2 {
      0% {font-size: 1em}
      100% {font-size: 1.2em}
    }
    `
		result := shim(css, CONTENT_ATTR, HOST_ATTR)
		animationLineRe := regexp.MustCompile(`animation:[^;]+;`)
		animationLineMatch := animationLineRe.FindString(result)
		animationLine := ""
		if animationLineMatch != "" {
			animationLine = animationLineMatch
		}
		scoped := []string{"my-anim", "my-anim2"}
		for _, s := range scoped {
			expected := "_ngcontent-%COMP%_" + s
			if !strings.Contains(animationLine, expected) {
				t.Errorf("Expected animationLine to contain %q, got %q", expected, animationLine)
			}
		}
		nonScoped := []string{"my-anim3", "my-anim4"}
		for _, s := range nonScoped {
			if !strings.Contains(animationLine, s) {
				t.Errorf("Expected animationLine to contain %q, got %q", s, animationLine)
			}
			if strings.Contains(animationLine, "_ngcontent-%COMP%_"+s) {
				t.Errorf("Expected animationLine to not contain '_ngcontent-%%COMP%%_%s', got %q", s, animationLine)
			}
		}
	})

	t.Run("should handle (scope or not) animation definitions preceded by an erroneous comma", func(t *testing.T) {
		COMPONENT_VARIABLE := "%COMP%"
		HOST_ATTR := "_nghost-" + COMPONENT_VARIABLE
		CONTENT_ATTR := "_ngcontent-" + COMPONENT_VARIABLE
		css := `.test {
      animation:, my-anim 1s,my-anim2 2s, my-anim3 3s,my-anim4 4s;
    }
    
    @keyframes my-anim {
      0% {color: red}
      100% {color: blue}
    }
    
    @keyframes my-anim2 {
      0% {font-size: 1em}
      100% {font-size: 1.2em}
    }
    `
		result := shim(css, CONTENT_ATTR, HOST_ATTR)
		if strings.Contains(result, "animation:,") {
			t.Errorf("Expected result to not contain 'animation:,', got %q", result)
		}
		animationLineRe := regexp.MustCompile(`animation:[^;]+;`)
		animationLineMatch := animationLineRe.FindString(result)
		animationLine := ""
		if animationLineMatch != "" {
			animationLine = animationLineMatch
		}
		scoped := []string{"my-anim", "my-anim2"}
		for _, s := range scoped {
			expected := "_ngcontent-%COMP%_" + s
			if !strings.Contains(animationLine, expected) {
				t.Errorf("Expected animationLine to contain %q, got %q", expected, animationLine)
			}
		}
		nonScoped := []string{"my-anim3", "my-anim4"}
		for _, s := range nonScoped {
			if !strings.Contains(animationLine, s) {
				t.Errorf("Expected animationLine to contain %q, got %q", s, animationLine)
			}
			if strings.Contains(animationLine, "_ngcontent-%COMP%_"+s) {
				t.Errorf("Expected animationLine to not contain '_ngcontent-%%COMP%%_%s', got %q", s, animationLine)
			}
		}
	})

	t.Run("should handle (scope or not) multiple animation definitions in a single declaration", func(t *testing.T) {
		css := `
        div {
            animation: 1s ease foo, 2s bar infinite, forwards baz 3s;
        }

        p {
            animation: 1s "foo", 2s "bar";
        }

        span {
            animation: .5s ease 'quux',
                        1s foo infinite, forwards "baz'" 1.5s,
                        2s bar;
        }

        button {
            animation: .5s bar,
                        1s foo 0.3s, 2s quux;
        }

        @keyframes bar {}
        @keyframes quux {}
        @keyframes "baz'" {}`
		result := shim(css, "host-a")
		if !strings.Contains(result, "animation: 1s ease foo, 2s host-a_bar infinite, forwards baz 3s;") {
			t.Errorf("Expected result to contain 'animation: 1s ease foo, 2s host-a_bar infinite, forwards baz 3s;'")
		}
		if !strings.Contains(result, "animation: 1s \"foo\", 2s \"host-a_bar\";") {
			t.Errorf("Expected result to contain 'animation: 1s \"foo\", 2s \"host-a_bar\";'")
		}
		if !strings.Contains(result, `
            animation: .5s host-a_bar,
                        1s foo 0.3s, 2s host-a_quux;`) {
			t.Errorf("Expected result to contain multi-line animation with host-a_bar and host-a_quux")
		}
		if !strings.Contains(result, `
            animation: .5s ease 'host-a_quux',
                        1s foo infinite, forwards "host-a_baz'" 1.5s,
                        2s host-a_bar;`) {
			t.Errorf("Expected result to contain multi-line animation with host-a_quux and host-a_baz'")
		}
	})

	t.Run("should not modify css variables ending with 'animation' even if they reference a local keyframes identifier", func(t *testing.T) {
		css := `
        button {
            --variable-animation: foo;
        }
        @keyframes foo {}`
		result := shim(css, "host-a")
		if !strings.Contains(result, "--variable-animation: foo;") {
			t.Errorf("Expected result to contain '--variable-animation: foo;', got %q", result)
		}
	})

	t.Run("should not modify css variables ending with 'animation-name' even if they reference a local keyframes identifier", func(t *testing.T) {
		css := `
        button {
            --variable-animation-name: foo;
        }
        @keyframes foo {}`
		result := shim(css, "host-a")
		if !strings.Contains(result, "--variable-animation-name: foo;") {
			t.Errorf("Expected result to contain '--variable-animation-name: foo;', got %q", result)
		}
	})

	t.Run("should maintain the spacing when handling (scoping or not) keyframes and animations", func(t *testing.T) {
		css := `
        div {
            animation-name : foo;
            animation:  5s bar   1s backwards;
            animation : 3s baz ;
            animation-name:foobar ;
            animation:1s "foo" ,   2s "bar",3s "quux";
        }

        @-webkit-keyframes  bar {}
        @keyframes foobar  {}
        @keyframes quux {}
        `
		result := shim(css, "host-a")
		if !strings.Contains(result, "animation-name : foo;") {
			t.Errorf("Expected result to contain 'animation-name : foo;'")
		}
		if !strings.Contains(result, "animation:  5s host-a_bar   1s backwards;") {
			t.Errorf("Expected result to contain 'animation:  5s host-a_bar   1s backwards;'")
		}
		if !strings.Contains(result, "animation : 3s baz ;") {
			t.Errorf("Expected result to contain 'animation : 3s baz ;'")
		}
		if !strings.Contains(result, "animation-name:host-a_foobar ;") {
			t.Errorf("Expected result to contain 'animation-name:host-a_foobar ;'")
		}
		if !strings.Contains(result, "@-webkit-keyframes  host-a_bar {}") {
			t.Errorf("Expected result to contain '@-webkit-keyframes  host-a_bar {}'")
		}
		if !strings.Contains(result, "@keyframes host-a_foobar  {}") {
			t.Errorf("Expected result to contain '@keyframes host-a_foobar  {}'")
		}
		if !strings.Contains(result, "animation:1s \"foo\" ,   2s \"host-a_bar\",3s \"host-a_quux\"") {
			t.Errorf("Expected result to contain 'animation:1s \"foo\" ,   2s \"host-a_bar\",3s \"host-a_quux\"'")
		}
	})

	t.Run("should correctly process animations defined without any prefixed space", func(t *testing.T) {
		testCases := []struct {
			css      string
			expected string
		}{
			{".test{display: flex;animation:foo 1s forwards;} @keyframes foo {}", ".test[host-a]{display: flex;animation:host-a_foo 1s forwards;} @keyframes host-a_foo {}"},
			{".test{animation:foo 2s forwards;} @keyframes foo {}", ".test[host-a]{animation:host-a_foo 2s forwards;} @keyframes host-a_foo {}"},
			{"button {display: block;animation-name: foobar;} @keyframes foobar {}", "button[host-a] {display: block;animation-name: host-a_foobar;} @keyframes host-a_foobar {}"},
		}

		for _, tc := range testCases {
			result := shim(tc.css, "host-a")
			if result != tc.expected {
				t.Errorf("For input %q, expected %q, got %q", tc.css, tc.expected, result)
			}
		}
	})

	t.Run("should correctly process keyframes defined without any prefixed space", func(t *testing.T) {
		testCases := []struct {
			css      string
			expected string
		}{
			{".test{display: flex;animation:bar 1s forwards;}@keyframes bar {}", ".test[host-a]{display: flex;animation:host-a_bar 1s forwards;}@keyframes host-a_bar {}"},
			{".test{animation:bar 2s forwards;}@-webkit-keyframes bar {}", ".test[host-a]{animation:host-a_bar 2s forwards;}@-webkit-keyframes host-a_bar {}"},
		}

		for _, tc := range testCases {
			result := shim(tc.css, "host-a")
			if result != tc.expected {
				t.Errorf("For input %q, expected %q, got %q", tc.css, tc.expected, result)
			}
		}
	})

	t.Run("should ignore keywords values when scoping local animations", func(t *testing.T) {
		css := `
        div {
            animation: inherit;
            animation: unset;
            animation: 3s ease reverse foo;
            animation: 5s foo 1s backwards;
            animation: none 1s foo;
            animation: .5s foo paused;
            animation: 1s running foo;
            animation: 3s linear 1s infinite running foo;
            animation: 5s foo ease;
            animation: 3s .5s infinite steps(3,end) foo;
            animation: 5s steps(9, jump-start) jump .5s;
            animation: 1s step-end steps;
        }

        @keyframes foo {}
        @keyframes inherit {}
        @keyframes unset {}
        @keyframes ease {}
        @keyframes reverse {}
        @keyframes backwards {}
        @keyframes none {}
        @keyframes paused {}
        @keyframes linear {}
        @keyframes running {}
        @keyframes end {}
        @keyframes jump {}
        @keyframes start {}
        @keyframes steps {}
        `
		result := shim(css, "host-a")
		expectedStrings := []string{
			"animation: inherit;",
			"animation: unset;",
			"animation: 3s ease reverse host-a_foo;",
			"animation: 5s host-a_foo 1s backwards;",
			"animation: none 1s host-a_foo;",
			"animation: .5s host-a_foo paused;",
			"animation: 1s running host-a_foo;",
			"animation: 3s linear 1s infinite running host-a_foo;",
			"animation: 5s host-a_foo ease;",
			"animation: 3s .5s infinite steps(3,end) host-a_foo;",
			"animation: 5s steps(9, jump-start) host-a_jump .5s;",
			"animation: 1s step-end host-a_steps;",
		}
		for _, expected := range expectedStrings {
			if !strings.Contains(result, expected) {
				t.Errorf("Expected result to contain %q, got %q", expected, result)
			}
		}
	})

	t.Run("should handle the usage of quotes", func(t *testing.T) {
		css := `
        div {
            animation: 1.5s foo;
        }

        p {
            animation: 1s 'foz bar';
        }

        @keyframes 'foo' {}
        @keyframes "foz bar" {}
        @keyframes bar {}
        @keyframes baz {}
        @keyframes qux {}
        @keyframes quux {}
        `
		result := shim(css, "host-a")
		if !strings.Contains(result, "@keyframes 'host-a_foo' {}") {
			t.Errorf("Expected result to contain '@keyframes 'host-a_foo' {}'")
		}
		if !strings.Contains(result, "@keyframes \"host-a_foz bar\" {}") {
			t.Errorf("Expected result to contain '@keyframes \"host-a_foz bar\" {}'")
		}
		if !strings.Contains(result, "animation: 1.5s host-a_foo;") {
			t.Errorf("Expected result to contain 'animation: 1.5s host-a_foo;'")
		}
		if !strings.Contains(result, "animation: 1s 'host-a_foz bar';") {
			t.Errorf("Expected result to contain 'animation: 1s 'host-a_foz bar';'")
		}
	})

	t.Run("should handle the usage of quotes containing escaped quotes", func(t *testing.T) {
		css := `
        div {
            animation: 1.5s "foo\"bar";
        }

        p {
            animation: 1s 'bar\' \'baz';
        }

        button {
            animation-name: 'foz " baz';
        }

        @keyframes "foo\"bar" {}
        @keyframes "bar' 'baz" {}
        @keyframes "foz \" baz" {}
        `
		result := shim(css, "host-a")
		if !strings.Contains(result, "@keyframes \"host-a_foo\\\"bar\" {}") {
			t.Errorf("Expected result to contain '@keyframes \"host-a_foo\\\"bar\" {}'")
		}
		if !strings.Contains(result, "@keyframes \"host-a_bar' 'baz\" {}") {
			t.Errorf("Expected result to contain '@keyframes \"host-a_bar' 'baz\" {}'")
		}
		if !strings.Contains(result, "@keyframes \"host-a_foz \\\" baz\" {}") {
			t.Errorf("Expected result to contain '@keyframes \"host-a_foz \\\" baz\" {}'")
		}
		if !strings.Contains(result, "animation: 1.5s \"host-a_foo\\\"bar\";") {
			t.Errorf("Expected result to contain 'animation: 1.5s \"host-a_foo\\\"bar\";'")
		}
		if !strings.Contains(result, "animation: 1s 'host-a_bar\\' \\'baz';") {
			t.Errorf("Expected result to contain 'animation: 1s 'host-a_bar\\' \\'baz';'")
		}
		if !strings.Contains(result, "animation-name: 'host-a_foz \" baz';") {
			t.Errorf("Expected result to contain 'animation-name: 'host-a_foz \" baz';'")
		}
	})

	t.Run("should handle the usage of commas in multiple animation definitions in a single declaration", func(t *testing.T) {
		css := `
         button {
           animation: 1s "foo bar, baz", 2s 'qux quux';
         }

         div {
           animation: 500ms foo, 1s 'bar, baz', 1500ms bar;
         }

         p {
           animation: 3s "bar, baz", 3s 'foo, bar' 1s, 3s "qux quux";
         }

         @keyframes "qux quux" {}
         @keyframes "bar, baz" {}
       `
		result := shim(css, "host-a")
		if !strings.Contains(result, "@keyframes \"host-a_qux quux\" {}") {
			t.Errorf("Expected result to contain '@keyframes \"host-a_qux quux\" {}'")
		}
		if !strings.Contains(result, "@keyframes \"host-a_bar, baz\" {}") {
			t.Errorf("Expected result to contain '@keyframes \"host-a_bar, baz\" {}'")
		}
		if !strings.Contains(result, "animation: 1s \"foo bar, baz\", 2s 'host-a_qux quux';") {
			t.Errorf("Expected result to contain 'animation: 1s \"foo bar, baz\", 2s 'host-a_qux quux';'")
		}
		if !strings.Contains(result, "animation: 500ms foo, 1s 'host-a_bar, baz', 1500ms bar;") {
			t.Errorf("Expected result to contain 'animation: 500ms foo, 1s 'host-a_bar, baz', 1500ms bar;'")
		}
		if !strings.Contains(result, "animation: 3s \"host-a_bar, baz\", 3s 'foo, bar' 1s, 3s \"host-a_qux quux\";") {
			t.Errorf("Expected result to contain 'animation: 3s \"host-a_bar, baz\", 3s 'foo, bar' 1s, 3s \"host-a_qux quux\";'")
		}
	})

	t.Run("should handle the usage of double quotes escaping in multiple animation definitions in a single declaration", func(t *testing.T) {
		css := `
        div {
            animation: 1s "foo", 1.5s "bar";
            animation: 2s "fo\"o", 2.5s "bar";
            animation: 3s "foo\"", 3.5s "bar", 3.7s "ba\"r";
            animation: 4s "foo\\", 4.5s "bar", 4.7s "baz\"";
            animation: 5s "fo\\\"o", 5.5s "bar", 5.7s "baz\"";
        }

        @keyframes "foo" {}
        @keyframes "fo\"o" {}
        @keyframes 'foo"' {}
        @keyframes 'foo\\' {}
        @keyframes bar {}
        @keyframes "ba\"r" {}
        @keyframes "fo\\\"o" {}
        `
		result := shim(css, "host-a")
		expectedStrings := []string{
			"@keyframes \"host-a_foo\" {}",
			"@keyframes \"host-a_fo\\\"o\" {}",
			"@keyframes 'host-a_foo\"' {}",
			"@keyframes 'host-a_foo\\\\' {}",
			"@keyframes host-a_bar {}",
			"@keyframes \"host-a_ba\\\"r\" {}",
			"@keyframes \"host-a_fo\\\\\\\"o\"",
			"animation: 1s \"host-a_foo\", 1.5s \"host-a_bar\";",
			"animation: 2s \"host-a_fo\\\"o\", 2.5s \"host-a_bar\";",
			"animation: 3s \"host-a_foo\\\"\", 3.5s \"host-a_bar\", 3.7s \"host-a_ba\\\"r\";",
			"animation: 4s \"host-a_foo\\\\\", 4.5s \"host-a_bar\", 4.7s \"baz\\\"\";",
			"animation: 5s \"host-a_fo\\\\\\\"o\", 5.5s \"host-a_bar\", 5.7s \"baz\\\"\";",
		}
		for _, expected := range expectedStrings {
			if !strings.Contains(result, expected) {
				t.Errorf("Expected result to contain %q, got %q", expected, result)
			}
		}
	})

	t.Run("should handle the usage of single quotes escaping in multiple animation definitions in a single declaration", func(t *testing.T) {
		css := `
        div {
            animation: 1s 'foo', 1.5s 'bar';
            animation: 2s 'fo\'o', 2.5s 'bar';
            animation: 3s 'foo\'', 3.5s 'bar', 3.7s 'ba\'r';
            animation: 4s 'foo\\', 4.5s 'bar', 4.7s 'baz\'';
            animation: 5s 'fo\\\'o', 5.5s 'bar', 5.7s 'baz\'';
        }

        @keyframes foo {}
        @keyframes 'fo\'o' {}
        @keyframes 'foo\'' {}
        @keyframes 'foo\\' {}
        @keyframes "bar" {}
        @keyframes 'ba\'r' {}
        @keyframes "fo\\\'o" {}
        `
		result := shim(css, "host-a")
		expectedStrings := []string{
			"@keyframes host-a_foo {}",
			"@keyframes 'host-a_fo\\'o' {}",
			"@keyframes 'host-a_foo\\'' {}",
			"@keyframes 'host-a_foo\\\\' {}",
			"@keyframes \"host-a_bar\" {}",
			"@keyframes 'host-a_ba\\'r' {}",
			"@keyframes \"host-a_fo\\\\\\'o\" {}",
			"animation: 1s 'host-a_foo', 1.5s 'host-a_bar';",
			"animation: 2s 'host-a_fo\\'o', 2.5s 'host-a_bar';",
			"animation: 3s 'host-a_foo\\'', 3.5s 'host-a_bar', 3.7s 'host-a_ba\\'r';",
			"animation: 4s 'host-a_foo\\\\', 4.5s 'host-a_bar', 4.7s 'baz\\'';",
			"animation: 5s 'host-a_fo\\\\\\'o', 5.5s 'host-a_bar', 5.7s 'baz\\''",
		}
		for _, expected := range expectedStrings {
			if !strings.Contains(result, expected) {
				t.Errorf("Expected result to contain %q, got %q", expected, result)
			}
		}
	})

	t.Run("should handle the usage of mixed single and double quotes escaping in multiple animation definitions in a single declaration", func(t *testing.T) {
		css := `
        div {
            animation: 1s 'f\"oo', 1.5s "ba\'r";
            animation: 2s "fo\"\"o", 2.5s 'b\\"ar';
            animation: 3s 'foo\\', 3.5s "b\\\"ar", 3.7s 'ba\'\"\'r';
            animation: 4s 'fo\'o', 4.5s 'b\"ar\"', 4.7s "baz\'";
        }

        @keyframes 'f"oo' {}
        @keyframes 'fo""o' {}
        @keyframes 'foo\\' {}
        @keyframes 'fo\'o' {}
        @keyframes 'ba\'r' {}
        @keyframes 'b\\"ar' {}
        @keyframes 'b\\\"ar' {}
        @keyframes 'b"ar"' {}
        @keyframes 'ba\'\"\'r' {}
        `
		result := shim(css, "host-a")
		expectedStrings := []string{
			"@keyframes 'host-a_f\"oo' {}",
			"@keyframes 'host-a_fo\"\"o' {}",
			"@keyframes 'host-a_foo\\\\' {}",
			"@keyframes 'host-a_fo\\'o' {}",
			"@keyframes 'host-a_ba\\'r' {}",
			"@keyframes 'host-a_b\\\\\"ar' {}",
			"@keyframes 'host-a_b\\\\\\\"ar' {}",
			"@keyframes 'host-a_b\"ar\"' {}",
			"@keyframes 'host-a_ba\\'\\\"\\'r' {}",
			"animation: 1s 'host-a_f\\\"oo', 1.5s \"host-a_ba\\'r\";",
			"animation: 2s \"host-a_fo\\\"\\\"o\", 2.5s 'host-a_b\\\\\"ar';",
			"animation: 3s 'host-a_foo\\\\', 3.5s \"host-a_b\\\\\\\"ar\", 3.7s 'host-a_ba\\'\\\"\\'r';",
			"animation: 4s 'host-a_fo\\'o', 4.5s 'host-a_b\\\"ar\\\"', 4.7s \"baz\\'\"",
		}
		for _, expected := range expectedStrings {
			if !strings.Contains(result, expected) {
				t.Errorf("Expected result to contain %q, got %q", expected, result)
			}
		}
	})

	t.Run("should handle the usage of commas inside quotes", func(t *testing.T) {
		css := `
        div {
            animation: 3s 'bar,, baz';
        }

        p {
            animation-name: "bar,, baz", foo,'ease, linear , inherit', bar;
        }

        @keyframes 'foo' {}
        @keyframes 'bar,, baz' {}
        @keyframes 'ease, linear , inherit' {}
        `
		result := shim(css, "host-a")
		if !strings.Contains(result, "@keyframes 'host-a_bar,, baz' {}") {
			t.Errorf("Expected result to contain '@keyframes 'host-a_bar,, baz' {}'")
		}
		if !strings.Contains(result, "animation: 3s 'host-a_bar,, baz';") {
			t.Errorf("Expected result to contain 'animation: 3s 'host-a_bar,, baz';'")
		}
		if !strings.Contains(result, "animation-name: \"host-a_bar,, baz\", host-a_foo,'host-a_ease, linear , inherit', bar;") {
			t.Errorf("Expected result to contain 'animation-name: \"host-a_bar,, baz\", host-a_foo,'host-a_ease, linear , inherit', bar;'")
		}
	})

	t.Run("should not ignore animation keywords when they are inside quotes", func(t *testing.T) {
		css := `
        div {
            animation: 3s 'unset';
        }

        button {
            animation: 5s "forwards" 1s forwards;
        }

        @keyframes unset {}
        @keyframes forwards {}
        `
		result := shim(css, "host-a")
		if !strings.Contains(result, "@keyframes host-a_unset {}") {
			t.Errorf("Expected result to contain '@keyframes host-a_unset {}'")
		}
		if !strings.Contains(result, "@keyframes host-a_forwards {}") {
			t.Errorf("Expected result to contain '@keyframes host-a_forwards {}'")
		}
		if !strings.Contains(result, "animation: 3s 'host-a_unset';") {
			t.Errorf("Expected result to contain 'animation: 3s 'host-a_unset';'")
		}
		if !strings.Contains(result, "animation: 5s \"host-a_forwards\" 1s forwards;") {
			t.Errorf("Expected result to contain 'animation: 5s \"host-a_forwards\" 1s forwards;'")
		}
	})

	t.Run("should handle css functions correctly", func(t *testing.T) {
		css := `
        div {
            animation: foo 0.5s alternate infinite cubic-bezier(.17, .67, .83, .67);
        }

        button {
            animation: calc(2s / 2) calc;
        }

        @keyframes foo {}
        @keyframes cubic-bezier {}
        @keyframes calc {}
        `
		result := shim(css, "host-a")
		if !strings.Contains(result, "@keyframes host-a_cubic-bezier {}") {
			t.Errorf("Expected result to contain '@keyframes host-a_cubic-bezier {}'")
		}
		if !strings.Contains(result, "@keyframes host-a_calc {}") {
			t.Errorf("Expected result to contain '@keyframes host-a_calc {}'")
		}
		if !strings.Contains(result, "animation: host-a_foo 0.5s alternate infinite cubic-bezier(.17, .67, .83, .67);") {
			t.Errorf("Expected result to contain 'animation: host-a_foo 0.5s alternate infinite cubic-bezier(.17, .67, .83, .67);'")
		}
		if !strings.Contains(result, "animation: calc(2s / 2) host-a_calc;") {
			t.Errorf("Expected result to contain 'animation: calc(2s / 2) host-a_calc;'")
		}
	})
}
