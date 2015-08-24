var TextField = React.createClass({displayName: "TextField",
  render: function() {
    return (
      React.createElement("div", {className: "form-group"}, 
        React.createElement("label", null, this.props.name), 
        React.createElement("input", {type: "text", className: "form-control", ref: this.props.name})
      )
    );
  }
});

var NumericalField = React.createClass({displayName: "NumericalField",
  render: function() {
    return (
      React.createElement("div", {className: "form-group"}, 
        React.createElement("label", null, this.props.name), 
        React.createElement("input", {type: "number", className: "form-control", min: this.props.minVal, step: this.props.interval, max: this.props.maxVal, ref: this.props.name})
      )
    );
  }
});

var BooleanField = React.createClass({displayName: "BooleanField",
  render: function() {
    return (
      React.createElement("div", null, 
        React.createElement("label", null, this.props.name), 
        React.createElement("br", null), 
        React.createElement("label", {className: "radio-inline"}, 
          React.createElement("input", {type: "radio", name: this.props.name, value: "false", checked: true}), " false"
        ), 
        React.createElement("label", {className: "radio-inline"}, 
          React.createElement("input", {type: "radio", name: this.props.name, value: "true"}), " true"
        )
      )
    );
  }
});

var PicklistField = React.createClass({displayName: "PicklistField",
  render: function() {
    var categories = [];
    this.props.categories.forEach(function(cat) {
      categories.push(React.createElement("option", null, cat));
    }.bind(this));
    return (
      React.createElement("div", null, 
        React.createElement("div", {className: "form-group"}, 
        React.createElement("label", null, this.props.name), 
        React.createElement("select", {ref: this.props.name, className: "form-control", multiple: this.props.isMultiple}, 
          categories
        )
        )
      )
    );
  }
});

var HTMLPreview = React.createClass({displayName: "HTMLPreview",
  handleSubmit: function(){
    console.log("submit triggered");
    $.ajax({
      type: "POST",
      data: {"form_type": "HTML", "form": $("#final-form-fields").html()},
      url: "/model-examples?model_id="+window.location.pathname.split("/")[2],
      dataType: 'json',
      cache: false,
      success: function(data) {
      }.bind(this),
      error: function(xhr, status, err) {
        console.error(this.props.url, status, err.toString());
      }.bind(this)
    });
  },

  render: function() {
    var jsxFields = [];
    if(this.props.fields.length > 0){

      this.props.fields.forEach(function(field) {
        if (field.dataType=="Text") {
          jsxFields.push(React.createElement(TextField, {name: field.name}));
        } else if (field.dataType=="Number") {
          jsxFields.push(React.createElement(NumericalField, {name: field.name, minVal: field.minVal, maxval: field.maxVal, interval: field.interval}));
        } else if (field.dataType=="Boolean") {
          jsxFields.push(React.createElement(BooleanField, {name: field.name}));
        } else if (field.dataType=="Picklist") {
          jsxFields.push(React.createElement(PicklistField, {name: field.name, categories: field.categories, isMultiple: field.isMultiple}));
        }
      }.bind(this));

      var finalForm = 
        React.createElement("form", {id: "final-form", onSubmit: this.handleSubmit}, 
          React.createElement("div", {id: "final-form-fields"}, 
            jsxFields
          ), 
        React.createElement("button", {id: "btn-save", className: "btn btn-info"}, "Save Form")
        )
    } else {
      var finalForm = "Add some form fields to the left to see a preview."
    }

    return (
      React.createElement("div", null, 
      finalForm
      )
    );
  }
});

var JSONPreview = React.createClass({displayName: "JSONPreview",
  render: function() {
    var sampleJSON = {};
    this.props.fields.forEach(function(field) {
      if (field.dataType=="Text") {
        sampleJSON[field.name] = "sample text";
      } else if (field.dataType=="Number") {
        var min = parseInt(field.minVal || "0");
        var max = parseInt(field.maxVal || "100");
        sampleJSON[field.name] = Math.floor(Math.random() * max + min)
      } else if (field.dataType=="Picklist") {
        sampleJSON[field.name] = field.categories[0];
      } else if (field.dataType=="Boolean") {
        sampleJSON[field.name] = false;
      }
    }.bind(this));
    return (
      React.createElement("pre", null, JSON.stringify(sampleJSON, null, 2))
    );
  }
});

var FieldPicker = React.createClass({displayName: "FieldPicker",
  getInitialState: function() {
    return { dataType: null };
  },
  handleChange: function(e) {
    this.setState({ dataType: this.refs.dataType.getDOMNode().value })
  },
  handleSubmit: function(e) {
    e.preventDefault();
    var field = {
      name: this.refs.fieldName.getDOMNode().value,
      dataType: this.state.dataType
    }
    if (field.dataType=="Number") {
      field.minVal = this.refs.minVal.getDOMNode().value;
      field.maxVal = this.refs.maxVal.getDOMNode().value;
      field.interval = this.refs.interval.getDOMNode().value;
    } else if (field.dataType=="Picklist") {
      field.categories = this.refs.picklistCategories.getDOMNode().value.trim().split('\n');
      field.isMultiple = $('input[name="nSelectionsOptions"]:checked').val()=="multiple";
    }
    this.props.addField(field);
    // reset the form
    this.refs.fieldName.getDOMNode().value = '';
    this.refs.dataType.getDOMNode().value = 'Please select a data type';
  },
  render: function() {
    var dataTypeFields;
    var submitButton = (React.createElement("button", {type: "submit", className: "btn btn-primary"}, "Add"));
    if (this.state.dataType=="Please select a data type") {
      submitButton = null;
    } else if (this.state.dataType=="Number") {
      dataTypeFields = (
        React.createElement("div", null, 
          React.createElement("div", {className: "form-group"}, 
            React.createElement("label", {for: "minVal"}, "Minimum Value"), 
            React.createElement("input", {type: "number", className: "form-control", ref: "minVal", placeholder: "No Minimum"})
          ), 
          React.createElement("div", {className: "form-group"}, 
            React.createElement("label", {for: "maxVal"}, "Maximum Value"), 
            React.createElement("input", {type: "number", className: "form-control", ref: "maxVal", placeholder: "No Maximum"})
          ), 
          React.createElement("div", {className: "form-group"}, 
            React.createElement("label", {for: "interval"}, "Interval"), 
            React.createElement("input", {type: "number", className: "form-control", ref: "interval", defaultValue: "1"})
          )
        )
      );
    } else if (this.state.dataType=="Picklist") {
      var exampleCategories = "Category One\nCategory Two\nCategory Three"
      dataTypeFields = (
        React.createElement("div", null, 
          React.createElement("div", {className: "form-group"}, 
            React.createElement("label", {for: "picklistCategories"}, "Picklist Categories"), 
            React.createElement("textarea", {ref: "picklistCategories", className: "form-control", rows: "10", placeholder: exampleCategories}
            )
          ), 
          React.createElement("div", {className: "form-group"}, 
            React.createElement("label", {for: "nSelections"}, "Number of Selections"), 
            React.createElement("br", null), 
            React.createElement("label", {className: "radio-inline"}, 
              React.createElement("input", {type: "radio", name: "nSelectionsOptions", ref: "nSelectionsOptions", value: "single", defaultChecked: true}), " Single"
            ), 
            React.createElement("label", {className: "radio-inline"}, 
              React.createElement("input", {type: "radio", name: "nSelectionsOptions", ref: "nSelectionsOptions", value: "multiple"}), " Multiple"
            )
          )
        )
      );
    }

    return (
      React.createElement("form", {onSubmit: this.handleSubmit}, 
        React.createElement("div", {className: "form-group"}, 
          React.createElement("label", {for: "fieldName"}, "Field Name"), 
          React.createElement("input", {type: "text", className: "form-control", name: "fieldName", ref: "fieldName", placeholder: "variable name", required: true})
        ), 
        React.createElement("div", {className: "form-group"}, 
          React.createElement("label", {for: "dataType"}, "Data Type"), 
          React.createElement("select", {className: "form-control", ref: "dataType", name: "dataType", onChange: this.handleChange}, 
            React.createElement("option", null, "Please select a data type"), 
            React.createElement("option", null, "Number"), 
            React.createElement("option", null, "Boolean"), 
            React.createElement("option", null, "Picklist"), 
            React.createElement("option", null, "Text")
          )
        ), 
        dataTypeFields, 
        submitButton
      )
    );
  }
});

var FormBuilder = React.createClass({displayName: "FormBuilder",
  getInitialState: function() {
    return { fields: [] };
  },
  addField: function(field) {
    this.setState({
      fields: this.state.fields.concat([field])
    });
  },
  render: function() {
    return (
    React.createElement("div", {className: "row"}, 
      React.createElement("div", {className: "col-sm-4"}, 
        React.createElement(FieldPicker, {addField: this.addField})
      ), 
      React.createElement("div", {className: "col-sm-7 col-sm-offset-1"}, 
        React.createElement(HTMLPreview, {fields: this.state.fields}), 
        React.createElement("hr", null), 
        React.createElement(JSONPreview, {fields: this.state.fields})
      )
    )
    );
  }
});

React.render(React.createElement(FormBuilder, null), document.getElementById("form-builder"));
React.render(React.createElement(ModelHeader, {url: "/models/1/json"}), document.getElementById("model-header"));

