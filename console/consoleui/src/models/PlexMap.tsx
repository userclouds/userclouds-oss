import Provider from './Provider';
import LoginApp from './LoginApp';
import Policy from './Policy';
import TelephonyProvider from './TelephonyProvider';

type PlexMap = {
  providers: Provider[];
  apps: LoginApp[];
  employee_app: LoginApp;
  policy: Policy;
  telephony_provider: TelephonyProvider;
  email_host: string;
  email_port: number;
  email_username: string;
  email_password: string;
};

export default PlexMap;
