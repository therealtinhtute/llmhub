export type RuntimeRoutingStrategy = 'round-robin' | 'weighted-round-robin' | 'fill-first';

export type RuntimeCredentialWeight = {
  auth_id: string;
  provider: string;
  weight: number;
};

export type RuntimeCredentialRoutingSettings = {
  strategy: RuntimeRoutingStrategy | string;
  weights: RuntimeCredentialWeight[];
};

export type RuntimeCloakingSettings = {
  disable_claude_model_list: boolean;
  disable_codex: boolean;
};

export type RuntimeICEServer = {
  urls: string[];
  username?: string;
};

export type RuntimeCodexLiveSettings = {
  enabled: boolean;
  max_sessions: number;
  disable_private_remote_ips: boolean;
  public_ip?: string;
  udp_port_min: number;
  udp_port_max: number;
  ice_servers: RuntimeICEServer[];
};

export type RuntimeHomeSettings = {
  enabled: boolean;
  disable_cluster_discovery: boolean;
};

export type RuntimeControlSettings = {
  revision: number;
  credential_routing: RuntimeCredentialRoutingSettings;
  cloaking: RuntimeCloakingSettings;
  codex_live: RuntimeCodexLiveSettings;
  home: RuntimeHomeSettings;
  cooldown_persistence_enabled: boolean;
};
