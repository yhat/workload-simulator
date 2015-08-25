Highcharts.setOptions({
    global: {
                useUTC: false
            },
colors: ['#26ADE4']
});


function QueryGraph(title)
{
    var _this = this;

    _this.initialize = function(title, config) {
        _this.config = $.extend(true, {}, config);
        console.log(_this.config);
        _this.config.title.text = title;
        _this.config.chart.events.load = function() { };

        _this.chart = new Highcharts.Chart(_this.config);

        _this.reinit_series();
    }

    _this.reinit_series = function() {
        var _this = this;
        _this.chart.series[0].setData([]);
    };

    _this.record_point = function(y) {
        var x = (new Date()).getTime();
        if (livemode && _this.chart.series[0].yData.length >= 1000)
        {
            _this.chart.series[0].addPoint([x, y], true, true);
        }
        else
        {
            _this.chart.series[0].addPoint([x, y]);
        }
    }

    _this.initialize(title);
}
