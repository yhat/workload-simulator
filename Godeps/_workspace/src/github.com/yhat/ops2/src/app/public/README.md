# ScienceOps public assets

## Compiling css from less

CSS compiles from the `less` directory to `static/css/styles.css`. To compile
the css simply run the `make` command.

```
# requires lessc
make css
```

## Compiling js from jsx

js compiles from the `jsx` directory to `static/js`. To compile the jsx to js
simply run the `make` command.

```
# requires jsx
make js
```

## Compile HTML (from templates)

All pages are put through a similar HTML template to take care of the `<head>`
tag and side bar. To compile the pages from `pages` to `static` run the
following command.

```
# requires go
make html
```

## Watch the directory for changes

If you would like the static assets to automatically compile as you work, run
the watch script.

```
./watch.sh
```
