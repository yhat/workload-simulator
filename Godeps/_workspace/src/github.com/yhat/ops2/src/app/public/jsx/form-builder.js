var TextField = React.createClass({
  render: function() {
    return (
      <div className="form-group">
        <label>{this.props.name}</label>
        <input type="text" className="form-control" ref={this.props.name} />
      </div>
    );
  }
});

var NumericalField = React.createClass({
  render: function() {
    return (
      <div className="form-group">
        <label>{this.props.name}</label>
        <input type="number" className="form-control" min={this.props.minVal} step={this.props.interval} max={this.props.maxVal} ref={this.props.name} />
      </div>
    );
  }
});

var BooleanField = React.createClass({
  render: function() {
    return (
      <div>
        <label>{this.props.name}</label>
        <br />
        <label className="radio-inline">
          <input type="radio" name={this.props.name} value="false" checked/> false
        </label>
        <label className="radio-inline">
          <input type="radio" name={this.props.name} value="true" /> true
        </label>
      </div>
    );
  }
});

var PicklistField = React.createClass({
  render: function() {
    var categories = [];
    this.props.categories.forEach(function(cat) {
      categories.push(<option>{cat}</option>);
    }.bind(this));
    return (
      <div>
        <div className="form-group">
        <label>{this.props.name}</label>
        <select ref={this.props.name} className="form-control" multiple={this.props.isMultiple}>
          {categories}
        </select>
        </div>
      </div>
    );
  }
});

var HTMLPreview = React.createClass({
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
          jsxFields.push(<TextField name={field.name} />);
        } else if (field.dataType=="Number") {
          jsxFields.push(<NumericalField name={field.name} minVal={field.minVal} maxval={field.maxVal} interval={field.interval} />);
        } else if (field.dataType=="Boolean") {
          jsxFields.push(<BooleanField name={field.name} />);
        } else if (field.dataType=="Picklist") {
          jsxFields.push(<PicklistField name={field.name} categories={field.categories} isMultiple={field.isMultiple} />);
        }
      }.bind(this));

      var finalForm = 
        <form id="final-form" onSubmit={this.handleSubmit}>
          <div id="final-form-fields">
            {jsxFields}
          </div>
        <button id="btn-save" className="btn btn-info">Save Form</button>
        </form>
    } else {
      var finalForm = "Add some form fields to the left to see a preview."
    }

    return (
      <div>
      {finalForm}
      </div>
    );
  }
});

var JSONPreview = React.createClass({
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
      <pre>{JSON.stringify(sampleJSON, null, 2)}</pre>
    );
  }
});

var FieldPicker = React.createClass({
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
    var submitButton = (<button type="submit" className="btn btn-primary">Add</button>);
    if (this.state.dataType=="Please select a data type") {
      submitButton = null;
    } else if (this.state.dataType=="Number") {
      dataTypeFields = (
        <div>
          <div className="form-group">
            <label for="minVal">Minimum Value</label>
            <input type="number" className="form-control" ref="minVal" placeholder="No Minimum" />
          </div>
          <div className="form-group">
            <label for="maxVal">Maximum Value</label>
            <input type="number" className="form-control" ref="maxVal" placeholder="No Maximum" />
          </div>
          <div className="form-group">
            <label for="interval">Interval</label>
            <input type="number" className="form-control" ref="interval" defaultValue="1" />
          </div>
        </div>
      );
    } else if (this.state.dataType=="Picklist") {
      var exampleCategories = "Category One\nCategory Two\nCategory Three"
      dataTypeFields = (
        <div>
          <div className="form-group">
            <label for="picklistCategories">Picklist Categories</label>
            <textarea ref="picklistCategories" className="form-control" rows="10" placeholder={exampleCategories}>
            </textarea>
          </div>
          <div className="form-group">
            <label for="nSelections">Number of Selections</label>
            <br />
            <label className="radio-inline">
              <input type="radio" name="nSelectionsOptions" ref="nSelectionsOptions" value="single" defaultChecked/> Single
            </label>
            <label className="radio-inline">
              <input type="radio" name="nSelectionsOptions" ref="nSelectionsOptions" value="multiple" /> Multiple
            </label>
          </div>
        </div>
      );
    }

    return (
      <form onSubmit={this.handleSubmit}>
        <div className="form-group">
          <label for="fieldName">Field Name</label>
          <input type="text" className="form-control" name="fieldName" ref="fieldName" placeholder="variable name" required />
        </div>
        <div className="form-group">
          <label for="dataType">Data Type</label>
          <select className="form-control" ref="dataType" name="dataType" onChange={this.handleChange}>
            <option>Please select a data type</option>
            <option>Number</option>
            <option>Boolean</option>
            <option>Picklist</option>
            <option>Text</option>
          </select>
        </div>
        {dataTypeFields}
        {submitButton}
      </form>
    );
  }
});

var FormBuilder = React.createClass({
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
    <div className="row">
      <div className="col-sm-4">
        <FieldPicker addField={this.addField} />
      </div>
      <div className="col-sm-7 col-sm-offset-1">
        <HTMLPreview fields={this.state.fields} />
        <hr />
        <JSONPreview fields={this.state.fields} />
      </div>
    </div>
    );
  }
});

React.render(<FormBuilder />, document.getElementById("form-builder"));
React.render(<ModelHeader url="/models/1/json" />, document.getElementById("model-header"));

