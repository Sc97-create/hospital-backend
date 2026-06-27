# Role Changes
## Need to change whole implementation of role, departments and role_permissions
## remove from createadminprof
## add it to organisation creation with orgid in it
# hold on createAdminProf
# prescription by organisation_id

## current focus
# getpatient by id for doctor
# add supplier
# add medicine
# auto import medicine from third party website
# add prescription
# Patient Preview
# Prescription Preview
# bedlisting page => free beds need to display
# logger
# full testing by sending to all friends
# stage deployment


## future focus
# dashboard revamp
# billing
# labtest
# report
# bp monitoring
# low code for patient to book from home
# sending code to given mobileno through msg


# api required for suppliers  => suppliers
  # getsuppliers
  # add supplier
  # delete supplier
  # deactivate supplier
  # search/sort/filter supplier in getsuppliers only

# api for prescription
  # add prescription    => payload = {medicine_id, patient_id, prescribed_by, frequency, dosage, duration, tablet-form, food-instruction,auto-generated-code,org_id}
  # preview prescription => 
  # search prescription
  # get prescription  => response = {status}
  # get prescription by id
  # data model := { id, medicine_id, patient_id, prescribed_by, frequency, dosage, duration, status, tablet-form, food-instruction,auto-generated-code,org_id, created_at, updated_at, deleted_at}


# medicine mental model what data comes to backend
  # supplier_id
  # invoice_date
  # medicine_array =>
    # medicine_name
    # form
    # strength
    # batch_no => auto creates in backend
    # expiry_date
    # quantity
    # unit_price
    # mrp

  # medicine_inventory
    id
    medicine_id
    supplier_id
    batch_no
    expiry_date
    purchase_qty 
    available_qty same as purchase_qty
    gst_pervent
    gst_amount
    total_amount
    c_at
    u_at
  # medicine
    id
    medicine_name
    form
    strength
    c_at
    u_at
  # medicine_stock_movements
    id
    medicine_inventory_id
    movmnt_type   "added","sold","expired"
    quantity_change
    c_at
    u_at
  # patient medicine order
    id
    patient_id
    prescription_id
    pharma_remark
    status
    c_at
    u_at
# operations to be followed
   add supplier
      gst_number
      gst_percentage is mandatory
   add medicine
        track all the medicines
        if the medicine is already present then just update the stock => how to do need to check
    medicine_inventory
        for every medicine create medicine_inventory
        when medicine is handed over to patient and pharmacy marks it done then change the available_qty in medicine_inventory
    medicine stock movemnts
       when pharma marks it as done, then create one record as issused
    patient medicine order items
        

 ###########################
 # polyclinic focus  => july 1st week
 # sstencil based prescription
 # make sure everything working out
 # sending notification for consultation, recurring appointments, tablet reminder
 # billing for consultation, prescription in one platform using cashfree third party api's
 # idea for multi lingual notification
 # testing for complete application
 # deployment


 #### appointment creation/ recurring events handling
 1) create appointment
 2) list appointment by patientid and organisationID
 3) edit appointment
 4) recurring event adding for appoitnment

   ## db schema
   ## appointments
   ## {
    id
    series_id
    patient_id
    doctor_id
    organisation_id
    appointment_date
    visit_type
    status
    created_at
   }
   ## notification_logs
   ## {
      id
      appointment_id
      channel   => [whatsapp, meta]
      status  => [failed, sent]
      sent_at => time when we have sent
      retry_count => need to decide based on reqmnt
   }
   ## appointment_series
   {
    id
    created_appointment_id
    patient_id
    doctor_id
    recurrence_type  => [Daily,weekly,monthly]
    occurence => 2,3,4
    days_of_week    => [mon,tue,wed and so on]
    start_date
    end_date
    status
    suggested_by
    created_at
   }

   # use cases
   # doctor said come every week on mon
      # recurrence type=> weekly
      # days_of_week => [mon]
      # start_date => suppose if today is sunday then nearest next is monday we should not schedule => next monday, same goes for all
      # if today is monday then next monday is next schedule for 1 or 2
      # end date based on occurence
   # doctor said come every 15days then
      # chose custom and added after 15days send event
      # we can capture that date and add it as start date and end date
   # doctor said come every month
      # check todays date, add date and get next month date for same day
      # ask for occurence here as well
   # appointments
      # for normal appointments
      # for series appointments => fetch it from appointment_series, get start date, end date and based on that create appointments, if daily then add date
      # if weekly then start date, end date and write logic for weekly
      # if monthly same take start date and end date and write logic for monthly
## clinic timings module
  ## db schema
  ## api calls
  ## 
  {
    id
    start_time
    end_time
    leave_days
    organisation_id
    created_at
    updated_at
  }


  ## compliance
  # HIPAA, HL7, and GDPR
  # health education
  # Pay-Per-Use Model
  # for fertility, skin based issue and mental health we can introduce telehealth as well
  # sending medicine on mobile no, by creating appointment through video call
  # connecting ayurvedic doctors and prescribing home medicines

appointment preview


  #
  Patient Information
  patient_name
  age
  gender
  mobile_number
  last_visit
  total_visit
  #
  #
  Assigned Doctor
  department
  visit type
  schedule date and time
  slot duration
  #
  #
  medicine history
  created_at
  id
  diagnosis
  prescription

  view_details=> 
  #

  `
 query:= `select a.appointment_code, a.id as appointment_id,a.start_time,a.end_time,a.appointment_date, pa.name, pa.age,pa.gender,pa.mobile_number, pa.last_visit,pr.created_at, pr.medicines as medicines from appointemnts as a
 join patients as pa on a.patient_id=p.id
 join users as u on a.doctor_id,u.id
 join departments as d on a.department_id=d.id
 join prescriptions as pr on a.prescription_id=p.id
 where organisation_id = $1
  `