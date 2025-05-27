type ServerContext = {
  ip_address: string;
  claims: Record<string, any>;
  action: string;
};

export default ServerContext;
