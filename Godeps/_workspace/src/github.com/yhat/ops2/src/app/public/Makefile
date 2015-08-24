LESS_FILES=$(shell find less -name '*.less' -type f)
JS_FILES=$(shell find jsx/*.js | sed 's/jsx/static\/js/')
HTML_FILES=$(shell find pages -name '*.html' -type f | sed 's/pages/static/')

.PHONEY: all css js html

all: css js html

css: static/css/styles.css

static/css/styles.css: $(LESS_FILES)
	lessc less/styles.less > static/css/styles.css

js: $(JS_FILES)

$(JS_FILES): $(shell find jsx -name '*.js' -type f)
	jsx --no-cache-dir jsx static/js

html: $(HTML_FILES)

static/%.html: pages/%.html base.tmpl
	go run build.go base.tmpl pages/$*.html > static/$*.html
