React.render(React.createElement(ModelHeader, {url: window.location.pathname + "/json", name: "BeerRec", lang: "python"}), document.getElementById("model-header"));
React.render(React.createElement(ModelNav, {page: "scoring", route: window.location.pathname}), document.getElementById("model-nav"));
