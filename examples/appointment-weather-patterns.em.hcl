# Appointment scheduling with state change, state view, automation, and translation workflows.

actor "scheduler" {
  title         = "Scheduler"
  auth_required = true
}

actor "calendar_user" {
  title         = "Calendar user"
  auth_required = true
}

system "weather_provider" {
  title    = "Weather provider"
  external = true
}

bounded_context "appointments" {
  title = "Appointments"

  aggregate "appointment" {
  }

  field_type "appointment_id" {
    type         = "UUID"
    id_attribute = true
  }

  field_type "starts_at" {
    type = "DateTime"
  }

  event "appointment_added" {
    title     = "Appointment Added"
    aggregate = aggregate.appointment

    field "appointment_id" {
      type = field_type.appointment_id
    }

    field "starts_at" {
      type = field_type.starts_at
    }
  }
}

bounded_context "weather" {
  title = "Weather"

  aggregate "weather_forecast" {
  }

  field_type "forecast" {
    type = "String"
  }

  field_type "changed_at" {
    type = "DateTime"
  }

  event "weather_predicted_for_appointment" {
    title     = "Weather predicted for appointment"
    aggregate = aggregate.weather_forecast

    field "appointment_id" {
      type = field_type.appointments.appointment_id
    }

    field "forecast" {
      type = field_type.forecast
    }
  }

  event "updated_weather_prediction" {
    title     = "Updated Weather Prediction"
    aggregate = aggregate.weather_forecast

    field "appointment_id" {
      type = field_type.appointments.appointment_id
    }

    field "forecast" {
      type = field_type.forecast
    }
  }
}

bounded_context "weather_provider" {
  title    = "Weather Provider"
  owner    = system.weather_provider
  external = true

  event "weather_forecast_changed" {
    title = "Weather Forecast Changed"

    field "appointment_id" {
      type = field_type.appointments.appointment_id
    }

    field "forecast" {
      type = field_type.weather.forecast
    }

    field "changed_at" {
      type = field_type.weather.changed_at
    }
  }
}

state_change "schedule_appointment" {
  title = "Schedule Appointment"

  screen "schedule_appointment_ui" {
    title = "UI"
    actor = actor.scheduler
    to    = [command.add_appointment]
  }

  screen "post_appointment" {
    title = "POST /appointment"
    to    = [command.add_appointment]
  }

  command "add_appointment" {
    title     = "Add Appointment"
    aggregate = aggregate.appointments.appointment
    to        = [event.appointments.appointment_added]

    field "appointment_id" {
      type = field_type.appointments.appointment_id
    }

    field "starts_at" {
      type = field_type.appointments.starts_at
    }
  }

  scenario "appointment_is_scheduled" {
    title = "Schedule an appointment"

    when {
      title   = "Add Appointment"
      command = command.add_appointment

      field "appointment_id" {
        type = field_type.appointments.appointment_id
      }

      field "starts_at" {
        type = field_type.appointments.starts_at
      }
    }

    then {
      title = "Appointment Added"
      event = event.appointments.appointment_added

      field "appointment_id" {
        type = field_type.appointments.appointment_id
      }

      field "starts_at" {
        type = field_type.appointments.starts_at
      }
    }
  }
}

state_view "view_calendar" {
  title = "View Calendar"

  readmodel "calendar" {
    title    = "Calendar"
    question = "Which appointments are on the calendar?"
    from     = [event.appointments.appointment_added]
    to       = [screen.calendar_ui, screen.get_calendar]

    field "appointment_id" {
      type = field_type.appointments.appointment_id
    }

    field "starts_at" {
      type = field_type.appointments.starts_at
    }
  }

  screen "calendar_ui" {
    title = "UI"
    actor = actor.calendar_user
  }

  screen "get_calendar" {
    title = "GET /calendar"
  }

  scenario "calendar_shows_scheduled_appointment" {
    title = "Show a scheduled appointment on the calendar"

    given {
      title = "Appointment Added"
      event = event.appointments.appointment_added

      field "appointment_id" {
        type = field_type.appointments.appointment_id
      }
    }

    then {
      title     = "Calendar"
      readmodel = readmodel.calendar

      field "appointment_id" {
        type = field_type.appointments.appointment_id
      }

      field "starts_at" {
        type = field_type.appointments.starts_at
      }
    }
  }
}

automation "add_weather_forecast" {
  title = "Add Weather Forecast"

  readmodel "appointments_without_weather_forecast" {
    title    = "Appointments without weather forecast"
    question = "Which appointments do not have a weather forecast?"
    from     = [event.appointments.appointment_added]
    to       = [processor.weather_processor]

    field "appointment_id" {
      type = field_type.appointments.appointment_id
    }

    field "starts_at" {
      type = field_type.appointments.starts_at
    }
  }

  processor "weather_processor" {
    title = "Weather Processor"
    to    = [command.add_weather_forecast_command]
  }

  command "add_weather_forecast_command" {
    title     = "Add Weather Forecast"
    aggregate = aggregate.weather.weather_forecast
    to        = [event.weather.weather_predicted_for_appointment]

    field "appointment_id" {
      type = field_type.appointments.appointment_id
    }

    field "forecast" {
      type = field_type.weather.forecast
    }
  }

  scenario "forecast_is_added_for_new_appointment" {
    title = "Add a weather forecast for a scheduled appointment"

    given {
      title = "Appointment Added"
      event = event.appointments.appointment_added

      field "appointment_id" {
        type = field_type.appointments.appointment_id
      }

      field "starts_at" {
        type = field_type.appointments.starts_at
      }
    }

    given {
      title     = "Appointments without weather forecast"
      readmodel = readmodel.appointments_without_weather_forecast
    }

    when {
      title     = "Weather Processor"
      processor = processor.weather_processor
    }

    then {
      title = "Weather predicted for appointment"
      event = event.weather.weather_predicted_for_appointment

      field "appointment_id" {
        type = field_type.appointments.appointment_id
      }

      field "forecast" {
        type = field_type.weather.forecast
      }
    }
  }
}

translation "translate_weather_change" {
  title = "Translate Changed Weather"

  readmodel "changed_predictions" {
    title    = "Changed Predictions"
    question = "Which external weather predictions changed?"
    from     = [event.weather_provider.weather_forecast_changed]
    to       = [processor.translator]

    field "appointment_id" {
      type = field_type.appointments.appointment_id
    }

    field "forecast" {
      type = field_type.weather.forecast
    }
  }

  processor "translator" {
    title = "Translator"
    to    = [command.translate_changed_weather]
  }

  command "translate_changed_weather" {
    title     = "Translate Changed Weather"
    aggregate = aggregate.weather.weather_forecast
    to        = [event.weather.updated_weather_prediction]

    field "appointment_id" {
      type = field_type.appointments.appointment_id
    }

    field "forecast" {
      type = field_type.weather.forecast
    }
  }

  scenario "external_weather_change_is_translated" {
    title = "Translate a changed external weather forecast"

    given {
      title = "Weather Forecast Changed"
      event = event.weather_provider.weather_forecast_changed

      field "appointment_id" {
        type = field_type.appointments.appointment_id
      }

      field "forecast" {
        type = field_type.weather.forecast
      }
    }

    given {
      title     = "Changed Predictions"
      readmodel = readmodel.changed_predictions
    }

    when {
      title     = "Translator"
      processor = processor.translator
    }

    then {
      title = "Updated Weather Prediction"
      event = event.weather.updated_weather_prediction

      field "appointment_id" {
        type = field_type.appointments.appointment_id
      }

      field "forecast" {
        type = field_type.weather.forecast
      }
    }
  }
}
