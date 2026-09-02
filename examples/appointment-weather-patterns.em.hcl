# Appointment scheduling with calendar views, an internal weather automation,
# and a translation of weather changes from an external provider.

slice "schedule-appointment" {
  title      = "Schedule Appointment"
  context    = "Command Pattern"
  slice_type = "STATE_CHANGE"
  aggregates = ["Appointments"]

  screen "schedule-appointment-ui" {
    title = "UI"
    type  = "SCREEN"

    dependency "add-appointment" {
      type         = "OUTBOUND"
      title        = "Add Appointment"
      element_type = "COMMAND"
    }
  }

  screen "post-appointment" {
    title = "POST /appointment"
    type  = "SCREEN"

    dependency "add-appointment" {
      type         = "OUTBOUND"
      title        = "Add Appointment"
      element_type = "COMMAND"
    }
  }

  command "add-appointment" {
    title     = "Add Appointment"
    type      = "COMMAND"
    aggregate = "Appointments"

    field "appointment_id" {
      type         = "UUID"
      id_attribute = true
    }
    field "starts_at" {
      type = "DateTime"
    }

    dependency "appointment-added" {
      type         = "OUTBOUND"
      title        = "Appointment Added"
      element_type = "EVENT"
    }
  }

  event "appointment-added" {
    title     = "Appointment Added"
    type      = "EVENT"
    aggregate = "Appointments"

    field "appointment_id" {
      type         = "UUID"
      id_attribute = true
    }
    field "starts_at" {
      type = "DateTime"
    }

    dependency "add-appointment" {
      type         = "INBOUND"
      title        = "Add Appointment"
      element_type = "COMMAND"
    }
  }

  actor "Scheduler" {
    auth_required = true
  }
}

slice "view-calendar" {
  title      = "View Calendar"
  context    = "View Pattern"
  slice_type = "STATE_VIEW"

  readmodel "calendar" {
    title = "Calendar"
    type  = "READMODEL"

    field "appointment_id" {
      type         = "UUID"
      id_attribute = true
    }
    field "starts_at" {
      type = "DateTime"
    }

    dependency "appointment-added" {
      type         = "INBOUND"
      title        = "Appointment Added"
      element_type = "EVENT"
    }
  }

  screen "calendar-ui" {
    title = "UI"
    type  = "SCREEN"

    dependency "calendar" {
      type         = "INBOUND"
      title        = "Calendar"
      element_type = "READMODEL"
    }
  }

  screen "get-calendar" {
    title = "GET /calendar"
    type  = "SCREEN"

    dependency "calendar" {
      type         = "INBOUND"
      title        = "Calendar"
      element_type = "READMODEL"
    }
  }

  actor "Calendar User" {
    auth_required = true
  }
}

slice "find-appointments-without-weather" {
  title      = "Appointments without weather forecast"
  context    = "Automation Pattern"
  slice_type = "STATE_VIEW"

  readmodel "appointments-without-weather-forecast" {
    title = "Appointments without weather forecast"
    type  = "READMODEL"

    field "appointment_id" {
      type         = "UUID"
      id_attribute = true
    }
    field "starts_at" {
      type = "DateTime"
    }

    dependency "appointment-added" {
      type         = "INBOUND"
      title        = "Appointment Added"
      element_type = "EVENT"
    }
  }
}

slice "add-weather-forecast" {
  title      = "Add Weather Forecast"
  context    = "Automation Pattern"
  slice_type = "AUTOMATION"
  aggregates = ["Weather"]

  processor "weather-processor" {
    title = "Weather Processor"
    type  = "AUTOMATION"

    dependency "appointments-without-weather-forecast" {
      type         = "INBOUND"
      title        = "Appointments without weather forecast"
      element_type = "READMODEL"
    }
    dependency "add-weather-forecast" {
      type         = "OUTBOUND"
      title        = "Add Weather Forecast"
      element_type = "COMMAND"
    }
  }

  command "add-weather-forecast" {
    title     = "Add Weather Forecast"
    type      = "COMMAND"
    aggregate = "Weather"

    field "appointment_id" {
      type         = "UUID"
      id_attribute = true
    }
    field "forecast" {
      type = "String"
    }

    dependency "weather-predicted-for-appointment" {
      type         = "OUTBOUND"
      title        = "Weather predicted for appointment"
      element_type = "EVENT"
    }
  }

  event "weather-predicted-for-appointment" {
    title     = "Weather predicted for appointment"
    type      = "EVENT"
    aggregate = "Weather"

    field "appointment_id" {
      type         = "UUID"
      id_attribute = true
    }
    field "forecast" {
      type = "String"
    }

    dependency "add-weather-forecast" {
      type         = "INBOUND"
      title        = "Add Weather Forecast"
      element_type = "COMMAND"
    }
  }

  actor "Weather Processor" {
    auth_required = false
  }
}

slice "translate-weather-change" {
  title      = "Translate Changed Weather"
  context    = "Translation Pattern"
  slice_type = "AUTOMATION"
  aggregates = ["Weather"]

  event "weather-forecast-changed" {
    title   = "Weather Forecast changed"
    type    = "EVENT"
    context = "EXTERNAL"

    field "appointment_id" {
      type         = "UUID"
      id_attribute = true
    }
    field "forecast" {
      type = "String"
    }
    field "changed_at" {
      type = "DateTime"
    }
  }

  readmodel "changed-predictions" {
    title = "Changed Predictions"
    type  = "READMODEL"

    field "appointment_id" {
      type         = "UUID"
      id_attribute = true
    }
    field "forecast" {
      type = "String"
    }

    dependency "weather-forecast-changed" {
      type         = "INBOUND"
      title        = "Weather Forecast changed"
      element_type = "EVENT"
    }
  }

  processor "translator" {
    title = "Translator"
    type  = "AUTOMATION"

    dependency "changed-predictions" {
      type         = "INBOUND"
      title        = "Changed Predictions"
      element_type = "READMODEL"
    }
    dependency "translate-changed-weather" {
      type         = "OUTBOUND"
      title        = "Translate Changed Weather"
      element_type = "COMMAND"
    }
  }

  command "translate-changed-weather" {
    title     = "Translate Changed Weather"
    type      = "COMMAND"
    aggregate = "Weather"

    field "appointment_id" {
      type         = "UUID"
      id_attribute = true
    }
    field "forecast" {
      type = "String"
    }

    dependency "updated-weather-prediction" {
      type         = "OUTBOUND"
      title        = "Updated weather prediction"
      element_type = "EVENT"
    }
  }

  event "updated-weather-prediction" {
    title     = "Updated weather prediction"
    type      = "EVENT"
    aggregate = "Weather"

    field "appointment_id" {
      type         = "UUID"
      id_attribute = true
    }
    field "forecast" {
      type = "String"
    }

    dependency "translate-changed-weather" {
      type         = "INBOUND"
      title        = "Translate Changed Weather"
      element_type = "COMMAND"
    }
  }

  actor "Translator" {
    auth_required = false
  }
}
