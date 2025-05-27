export type DataSource = {
  id: string;
  name: string;
  type: string;
  config: {
    host: string;
    port: number;
    database: string;
    username: string;
    password: string;
  };
  metadata: Record<string, any>;
};

export const dataSourceColumns = [
  'name',
  'type',
  'metadata->>format',
  'metadata->>classifications',
  'metadata->>storage',
  'metadata->>regulations',
];

export const dataSourcesPrefix = 'datasources_';

export const blankDataSource = () => ({
  id: '',
  name: '',
  type: 'file',
  config: {
    host: '',
    port: 0,
    database: '',
    username: '',
    password: '',
  },
  metadata: {
    description: '',
    contains_pii: false,
    format: [],
    classifications: [],
    regulations: [],
    storage: 'cloud',
    '3p_hosted': false,
    '3p_managed': false,
  },
});

export type DataSourceElement = {
  id: string;
  data_source_id: string;
  path: string;
  type: string;
  metadata: Record<string, any>;
};

export const dataSourceElementColumns = [
  'path',
  'type',
  'data_source_id',
  'metadata->>owner',
  'metadata->>contents',
  'metadata->>regulations',
  'metadata->>tags',
];

export const blankDataSourceElement = () => ({
  id: '',
  data_source_id: '',
  path: '',
  type: '',
  metadata: {
    contents: [],
    contains_pii: false,
    regulations: [],
    tags: [],
    owners: '',
  },
});

export enum DataFormats {
  structured = 'Structured',
  semistrucutred = 'Semi-structured',
  unstructured = 'Unstructured',
}

export enum DataClassifications {
  financial = 'Financial',
  health = 'Health/medical',
  geo = 'Location/geo',
  contact = 'Contact info',
  biographical = 'Biographical',
  biometric = 'Biometric',
  child = "Children's data",
  education = 'Education',
  sex = 'Gender/sex/sexuality',
}

export enum DataStorageOptions {
  cloud = 'Cloud',
  onprem = 'On-prem',
}

export enum Regulations {
  bipa = 'BIPA (Illinois)',
  ccpa = 'CCPA/CPRA (Calif.)',
  coppa = 'COPPA',
  fcra = 'FCRA/FACTA',
  ferpa = 'FERPA',
  gdpr = 'GDPR',
  gina = 'GINA',
  glba = 'GLBA',
  hipaa = 'HIPAA',
  tcpa = 'TCPA',
  vppa = 'VPPA',
}

export enum UserDataTypes {
  phone = 'Phone #',
  email = 'Email',
  address = 'Address',
  account_number = 'Account #',
  birthdate = 'Birthdate',
  height = 'Height',
  weight = 'Weight',
  eye_color = 'Eye color',
  age = 'Age',
  device_id = 'Device ID',
  IP_address = 'IP address',
  MAC_address = 'MAC address',
  sex = 'Sex',
  gender = 'Gender',
  orientation = 'Sexual orientation',
  race = 'Race/ethnicity',
  SSN = 'SSN',
  employee_id = 'Employee ID',
  password = 'Password',
  zip = 'ZIP code',
  government_id = 'Government ID #',
}

export enum DataSourceTypes {
  file = 'File',
  postgres = 'Postgres',
  redshift = 'Redshift',
  airflow = 'Airflow',
  spark = 'Spark',
  bigquery = 'BigQuery',
  snowflake = 'Snowflake',
  athena = 'Athena',
  oracle = 'Oracle',
  mongodb = 'MongoDB',
  dynamodb = 'DynamoDB',
  elasticsearch = 'Elasticsearch',
  kafka = 'Kafka',
  glue = 'Amazon Glue',
  fivetran = 'Fivetran',
  mysql = 'MySQL',
  tableau = 'Tableau',
  s3bucket = 'S3 bucket',
  other = 'Other',
}

export default DataSource;

export const dataSourceElementsPrefix = 'dselements_';

export const defaultJIRATicketOwner = 'will@userclouds.com';
