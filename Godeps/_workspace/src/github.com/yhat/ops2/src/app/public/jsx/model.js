React.render(<ModelHeader url={window.location.pathname + "/json"} name="BeerRec" lang="python" />, document.getElementById("model-header"));
React.render(<ModelNav page="scoring" route={window.location.pathname} />, document.getElementById("model-nav"));
