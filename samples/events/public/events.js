'use strict';

var AsyncTypeahead = ReactBootstrapTypeahead.AsyncTypeahead

const e = React.createElement;

function tryGetJSON(response) {
  if (!response.ok) {
      throw new Error(
          `fetch error (code: ${response.status}): ${response.statusText}`)
  }
  return response.json()
}

function makeEventsURL(path, query) {
  if (!query) {
    return path
  }

  // Since this is running on the same protocol & host, use relative paths.
  return path + "?" + new URLSearchParams(query).toString()
}

// Shared by EventRow and EventCreator
function onInviteeSearch(query) {
  this.setState({searching: true})
  var url = makeEventsURL("/api/users", {filter: query})
  fetch(url)
    .then(result => tryGetJSON(result))
    .then(
      (result) => {
        this.setState({
          searching: false,
          searchedPersons: result.filter(x => x.id != MyUserID),
        });
    })
    .catch((e) => {
      this.setState({
        searching: false})
    })
}

class EventRow extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      editing: false,
      submitting: false,
      searching: false,
      inputTitle: this.props.event.title,
      inputInvitees: [],
      fetchedPersons: [],
      searchedPersons: [],
    };
    this.onEditClick = this.onEditClick.bind(this)
    this.onDeleteClick = this.onDeleteClick.bind(this)
    this.onSaveClick = this.onSaveClick.bind(this)
    this.onCancelClick = this.onCancelClick.bind(this)
    this.onTitleChange = this.onTitleChange.bind(this)
    this.onInviteeSearch = onInviteeSearch.bind(this)
    this.onInviteeChange = this.onInviteeChange.bind(this)
  }

  onEditClick(event) {
    this.setState({editing: true})
  }
  onDeleteClick(event) {
    this.setState({submitting: true})
    let url = makeEventsURL("/api/event/" + this.props.event.id)
    fetch(url, { method: 'DELETE' })
      .then(
        (response) => {
          if (response.ok) {
            this.props.onEventChanged()
          }
        })
      .finally(
        () => this.setState({editing: false, submitting: false})
      )
    event.preventDefault();
  }

  onSaveClick(event) {
    this.setState({submitting: true})
    let url = makeEventsURL("/api/event/" + this.props.event.id)
    fetch(url, { method: 'PUT',
      body: JSON.stringify({
        title: this.state.inputTitle,
        invitees: this.state.inputInvitees})
      })
      .then(
        (response) => {
          if (response.ok) {
            this.props.onEventChanged()
          }
        })
      .finally(
        () => this.setState({editing: false, submitting: false})
      )
    event.preventDefault();
  }

  onCancelClick(event) {
    this.setState({editing: false, inputTitle: this.props.event.title})
  }

  onTitleChange(event) {
    this.setState({inputTitle: event.target.value})
  }

  onInviteeChange(invitees) {
    this.setState({inputInvitees: invitees})
  }

  componentDidMount() {
    this.refreshEvent()
  }

  componentDidUpdate(prevProps) {
    if (this.props.event !== prevProps.event) {
      this.refreshEvent()
    }
  }

  refreshEvent() {
    let query = "id=" + this.props.event.creator.id
    for (var i = 0; i < this.props.event.invitees.length; i++) {
      query = query + "&"
      query = query + "id=" + this.props.event.invitees[i].id
    }
    let personsUrl = makeEventsURL("/api/users", query)
    fetch(personsUrl)
      .then(res => res.json())
      .then(
        (result) => {
          this.setState({
            fetchedPersons: result,
            inputInvitees: result,
          });
        },
        // I don't know why but React examples recommend this pattern instead of catch?
        (error) => {
          this.setState({
            fetchedPersons: [],
          });
        }
      )
  }

  render() {
    let invitees = this.props.event.invitees.map((user) => {
      return this.state.fetchedPersons.find(p => p.id == user.id)
    }).filter(p => !!p)
    let creator = this.state.fetchedPersons.find(p => p.id == this.props.event.creator.id)
    let creatorName = creator ? creator.name : "[not loaded]"
    if (this.state.editing) {
      var textInputProps = {
        type: "text",
        disabled: this.state.submitting
      }
      var typeahead = e(AsyncTypeahead, {
        defaultSelected: invitees,
        filterBy: () => true,
        id: "invitees",
        allowNew: true,
        multiple: true,
        onSearch: this.onInviteeSearch,
        onChange: this.onInviteeChange,
        isLoading: this.state.searching,
        labelKey: "name",
        options: this.state.searchedPersons})

      return e('tr',{},
          e('td',{},e('input', {id: 'title_input', value: this.state.inputTitle, onChange: this.onTitleChange, ...textInputProps})),
          e('td',{},),
          e('td',{},typeahead),
          e('td',{},
            e('button', {className: "btn btn-primary", onClick: this.onSaveClick, disabled: this.state.submitting}, "Save"),
            e('button', {className: "btn btn-secondary", onClick: this.onCancelClick, disabled: this.state.submitting}, "Cancel")),
        )
    } else {
      let editButtons = [];
      if (this.props.event.creator.id == MyUserID) {
        editButtons.push(e('button', {className: "btn btn-primary", onClick: this.onEditClick}, "Edit"))
        editButtons.push(e('button', {className: "btn btn-warning", onClick: this.onDeleteClick}, "Delete"))
      }
      return e('tr',{},
          e('td',{}, this.props.event.title),
          e('td',{}, creatorName),
          e('td',{}, invitees.map(x => x.name).join(',')),
          e('td',{}, ...editButtons)
        )
    }
  }
}

class EventsTable extends React.Component {
  render() {
    const rows = [];
    this.props.events.forEach((event) => {
      rows.push(e(EventRow,{event: event, key: event.id, onEventChanged: this.props.onEventChanged},''))
    })
    return e(
      'table', {className: "table"},
        e('thead',{},
          e(
            'tr',{},
            e('th',{scope: "col"},"Title"),
            e('th',{scope: "col"},"Host"),
            e('th',{scope: "col"},"Invitees"),
            e('th',{scope: "col"},""),
          )
        ),
        e('tbody',{},rows)
      )
  }
}

class EventCreator extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      creating: false,
      submitting: false,
      searching: false,
      title: '',
      searchedPersons: [],
      invitees: [],
    };
    this.onCreateClick = this.onCreateClick.bind(this)
    this.onTitleChange = this.onTitleChange.bind(this)
    this.onSubmitClick = this.onSubmitClick.bind(this)
    this.onCancelClick = this.onCancelClick.bind(this)
    this.onInviteeSearch = onInviteeSearch.bind(this)
    this.onInviteeChange = this.onInviteeChange.bind(this)
  }

  onTitleChange(event) {
    this.setState({title: event.target.value})
  }

  onCreateClick() {
    this.setState({creating: true})
  }

  onSubmitClick(event) {
    this.setState({submitting: true})
    let url = makeEventsURL("/api/event")
    fetch(url, {
      method: 'POST',
      body: JSON.stringify({
        title: this.state.title,
        invitees: this.state.invitees,
      })})
      .then(
        (response) => {
          if (response.ok) {
            this.props.onEventCreated()
          }
        })
      .finally(
        () => this.setState({creating: false, submitting: false})
      )
    event.preventDefault();
  }

  onCancelClick() {
    this.setState({creating: false, submitting: false})
  }

  onInviteeChange(invitees) {
    this.setState({invitees: invitees})
  }

  render() {
    if (this.state.creating) {
      var textInputProps = {
        type: "text",
        disabled: this.state.submitting
      }
      var typeahead = e(AsyncTypeahead, {
        filterBy: () => true,
        id: "invitees",
        allowNew: true,
        multiple: true,
        onSearch: this.onInviteeSearch,
        onChange: this.onInviteeChange,
        isLoading: this.state.searching,
        labelKey: "name",
        options: this.state.searchedPersons})

      return e('form', {onSubmit: this.onSubmitClick},
        e('div',{},e('span',{},"Title: "),e('input', {id: 'title_input', value: this.state.title, onChange: this.onTitleChange, ...textInputProps})),
        e('div',{},e('span',{},"Invitees: "),typeahead),
        e('div',{},e('input', {type: "submit", className: "btn btn-primary"}), e('button', {className: "btn btn-secondary", onClick: this.onCancelClick}, "Cancel")),
      )
    } else {
      return e('button', {className: "btn btn-primary", onClick: this.onCreateClick}, "Create")
    }
  }
}

class EventsWidget extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      isLoaded: false,
      error: null,
      events: [],
    };

    this.onEventChanged = this.onEventChanged.bind(this)
  }

  refreshEvents() {
    let eventsUrl = makeEventsURL("/api/events")
    fetch(eventsUrl)
      .then((result) => {
        if (result.status == 404) {
          return []
        } else {
          return tryGetJSON(result)
        }
      })
      .then(
        (result) => {
          result.sort((a, b) => a.title.localeCompare(b.title))
          this.setState({
            isLoaded: true,
            events: result
          });
      })
      .catch((e) => {
        console.log(e.message)
          this.setState({
            isLoaded: true,
            error: e})
      })
  }

  componentDidMount() {
    this.refreshEvents()
  }

  onEventChanged() {
    this.refreshEvents()
  }

  render() {
    const { error, isLoaded, events } = this.state;
    if (error) {
      return e('div', {}, error.message)
    } else if (!isLoaded) {
      return e('div', {}, "Loading...")
    } else {
      return e('div',{}, e(EventsTable, {events: this.state.events, onEventChanged:this.onEventChanged}),
        e(EventCreator, {onEventCreated: this.onEventChanged}))
    }
  }
}

const domContainer = document.querySelector('#table_parent');
ReactDOM.render(e(EventsWidget), domContainer);
