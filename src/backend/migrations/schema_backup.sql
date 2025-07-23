--
-- PostgreSQL database dump
--

-- Dumped from database version 15.13
-- Dumped by pg_dump version 15.13

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: hosts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.hosts (
    id bigint NOT NULL,
    project_id bigint NOT NULL,
    name character varying(255) NOT NULL,
    ip character varying(45) NOT NULL,
    port bigint DEFAULT 22,
    "user" character varying(100) NOT NULL,
    "group" character varying(100),
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);


--
-- Name: hosts_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.hosts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: hosts_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.hosts_id_seq OWNED BY public.hosts.id;


--
-- Name: permissions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.permissions (
    id bigint NOT NULL,
    resource character varying(100) NOT NULL,
    verb character varying(50) NOT NULL,
    scope character varying(100) DEFAULT '*'::character varying,
    description character varying(500),
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);


--
-- Name: permissions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.permissions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: permissions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.permissions_id_seq OWNED BY public.permissions.id;


--
-- Name: playbook_generations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.playbook_generations (
    id bigint NOT NULL,
    project_id bigint NOT NULL,
    file_name character varying(255) NOT NULL,
    file_path character varying(500) NOT NULL,
    status character varying(50) DEFAULT 'success'::character varying,
    error text,
    created_at timestamp with time zone
);


--
-- Name: playbook_generations_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.playbook_generations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: playbook_generations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.playbook_generations_id_seq OWNED BY public.playbook_generations.id;


--
-- Name: projects; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.projects (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    name character varying(255) NOT NULL,
    description character varying(1000),
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);


--
-- Name: projects_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.projects_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: projects_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.projects_id_seq OWNED BY public.projects.id;


--
-- Name: role_bindings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.role_bindings (
    id bigint NOT NULL,
    name character varying(100) NOT NULL,
    role_id bigint NOT NULL,
    namespace character varying(100) DEFAULT 'default'::character varying,
    subject_type character varying(50) NOT NULL,
    subject_id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);


--
-- Name: role_bindings_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.role_bindings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: role_bindings_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.role_bindings_id_seq OWNED BY public.role_bindings.id;


--
-- Name: role_permissions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.role_permissions (
    id bigint NOT NULL,
    role_id bigint NOT NULL,
    permission_id bigint NOT NULL,
    created_at timestamp with time zone
);


--
-- Name: role_permissions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.role_permissions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: role_permissions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.role_permissions_id_seq OWNED BY public.role_permissions.id;


--
-- Name: roles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.roles (
    id bigint NOT NULL,
    name character varying(100) NOT NULL,
    description character varying(500),
    is_system boolean DEFAULT false,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);


--
-- Name: roles_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.roles_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: roles_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.roles_id_seq OWNED BY public.roles.id;


--
-- Name: tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tasks (
    id bigint NOT NULL,
    project_id bigint NOT NULL,
    name character varying(255) NOT NULL,
    module character varying(100) NOT NULL,
    args text,
    order_num bigint DEFAULT 0,
    enabled boolean DEFAULT true,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);


--
-- Name: tasks_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.tasks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: tasks_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.tasks_id_seq OWNED BY public.tasks.id;


--
-- Name: user_group_memberships; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_group_memberships (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    user_group_id bigint NOT NULL,
    created_at timestamp with time zone
);


--
-- Name: user_group_memberships_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.user_group_memberships_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_group_memberships_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.user_group_memberships_id_seq OWNED BY public.user_group_memberships.id;


--
-- Name: user_group_roles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_group_roles (
    id bigint NOT NULL,
    user_group_id bigint NOT NULL,
    role_id bigint NOT NULL,
    created_at timestamp with time zone
);


--
-- Name: user_group_roles_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.user_group_roles_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_group_roles_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.user_group_roles_id_seq OWNED BY public.user_group_roles.id;


--
-- Name: user_groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_groups (
    id bigint NOT NULL,
    name character varying(100) NOT NULL,
    description character varying(500),
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);


--
-- Name: user_groups_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.user_groups_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_groups_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.user_groups_id_seq OWNED BY public.user_groups.id;


--
-- Name: user_roles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_roles (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    role_id bigint NOT NULL,
    created_at timestamp with time zone
);


--
-- Name: user_roles_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.user_roles_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_roles_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.user_roles_id_seq OWNED BY public.user_roles.id;


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id bigint NOT NULL,
    username character varying(100) NOT NULL,
    email character varying(255) NOT NULL,
    password character varying(255) NOT NULL,
    is_active boolean DEFAULT true,
    last_login timestamp with time zone,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);


--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;


--
-- Name: variables; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.variables (
    id bigint NOT NULL,
    project_id bigint NOT NULL,
    name character varying(255) NOT NULL,
    value text,
    type character varying(50) DEFAULT 'string'::character varying,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);


--
-- Name: variables_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.variables_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: variables_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.variables_id_seq OWNED BY public.variables.id;


--
-- Name: hosts id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.hosts ALTER COLUMN id SET DEFAULT nextval('public.hosts_id_seq'::regclass);


--
-- Name: permissions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.permissions ALTER COLUMN id SET DEFAULT nextval('public.permissions_id_seq'::regclass);


--
-- Name: playbook_generations id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.playbook_generations ALTER COLUMN id SET DEFAULT nextval('public.playbook_generations_id_seq'::regclass);


--
-- Name: projects id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.projects ALTER COLUMN id SET DEFAULT nextval('public.projects_id_seq'::regclass);


--
-- Name: role_bindings id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_bindings ALTER COLUMN id SET DEFAULT nextval('public.role_bindings_id_seq'::regclass);


--
-- Name: role_permissions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_permissions ALTER COLUMN id SET DEFAULT nextval('public.role_permissions_id_seq'::regclass);


--
-- Name: roles id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.roles ALTER COLUMN id SET DEFAULT nextval('public.roles_id_seq'::regclass);


--
-- Name: tasks id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tasks ALTER COLUMN id SET DEFAULT nextval('public.tasks_id_seq'::regclass);


--
-- Name: user_group_memberships id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_group_memberships ALTER COLUMN id SET DEFAULT nextval('public.user_group_memberships_id_seq'::regclass);


--
-- Name: user_group_roles id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_group_roles ALTER COLUMN id SET DEFAULT nextval('public.user_group_roles_id_seq'::regclass);


--
-- Name: user_groups id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_groups ALTER COLUMN id SET DEFAULT nextval('public.user_groups_id_seq'::regclass);


--
-- Name: user_roles id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_roles ALTER COLUMN id SET DEFAULT nextval('public.user_roles_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- Name: variables id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.variables ALTER COLUMN id SET DEFAULT nextval('public.variables_id_seq'::regclass);


--
-- Name: hosts hosts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.hosts
    ADD CONSTRAINT hosts_pkey PRIMARY KEY (id);


--
-- Name: permissions permissions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.permissions
    ADD CONSTRAINT permissions_pkey PRIMARY KEY (id);


--
-- Name: playbook_generations playbook_generations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.playbook_generations
    ADD CONSTRAINT playbook_generations_pkey PRIMARY KEY (id);


--
-- Name: projects projects_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.projects
    ADD CONSTRAINT projects_pkey PRIMARY KEY (id);


--
-- Name: role_bindings role_bindings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_bindings
    ADD CONSTRAINT role_bindings_pkey PRIMARY KEY (id);


--
-- Name: role_permissions role_permissions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_permissions
    ADD CONSTRAINT role_permissions_pkey PRIMARY KEY (id);


--
-- Name: roles roles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_pkey PRIMARY KEY (id);


--
-- Name: tasks tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tasks
    ADD CONSTRAINT tasks_pkey PRIMARY KEY (id);


--
-- Name: user_group_memberships user_group_memberships_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_group_memberships
    ADD CONSTRAINT user_group_memberships_pkey PRIMARY KEY (id);


--
-- Name: user_group_roles user_group_roles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_group_roles
    ADD CONSTRAINT user_group_roles_pkey PRIMARY KEY (id);


--
-- Name: user_groups user_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_groups
    ADD CONSTRAINT user_groups_pkey PRIMARY KEY (id);


--
-- Name: user_roles user_roles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT user_roles_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: variables variables_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.variables
    ADD CONSTRAINT variables_pkey PRIMARY KEY (id);


--
-- Name: idx_hosts_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_hosts_deleted_at ON public.hosts USING btree (deleted_at);


--
-- Name: idx_hosts_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_hosts_project_id ON public.hosts USING btree (project_id);


--
-- Name: idx_playbook_generations_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_playbook_generations_project_id ON public.playbook_generations USING btree (project_id);


--
-- Name: idx_projects_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_projects_deleted_at ON public.projects USING btree (deleted_at);


--
-- Name: idx_projects_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_projects_user_id ON public.projects USING btree (user_id);


--
-- Name: idx_role_bindings_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_role_bindings_deleted_at ON public.role_bindings USING btree (deleted_at);


--
-- Name: idx_role_bindings_role_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_role_bindings_role_id ON public.role_bindings USING btree (role_id);


--
-- Name: idx_role_bindings_subject_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_role_bindings_subject_id ON public.role_bindings USING btree (subject_id);


--
-- Name: idx_role_permissions_permission_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_role_permissions_permission_id ON public.role_permissions USING btree (permission_id);


--
-- Name: idx_role_permissions_role_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_role_permissions_role_id ON public.role_permissions USING btree (role_id);


--
-- Name: idx_roles_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_roles_deleted_at ON public.roles USING btree (deleted_at);


--
-- Name: idx_roles_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_roles_name ON public.roles USING btree (name);


--
-- Name: idx_tasks_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tasks_deleted_at ON public.tasks USING btree (deleted_at);


--
-- Name: idx_tasks_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tasks_project_id ON public.tasks USING btree (project_id);


--
-- Name: idx_user_group_memberships_user_group_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_group_memberships_user_group_id ON public.user_group_memberships USING btree (user_group_id);


--
-- Name: idx_user_group_memberships_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_group_memberships_user_id ON public.user_group_memberships USING btree (user_id);


--
-- Name: idx_user_group_roles_role_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_group_roles_role_id ON public.user_group_roles USING btree (role_id);


--
-- Name: idx_user_group_roles_user_group_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_group_roles_user_group_id ON public.user_group_roles USING btree (user_group_id);


--
-- Name: idx_user_groups_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_groups_deleted_at ON public.user_groups USING btree (deleted_at);


--
-- Name: idx_user_groups_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_user_groups_name ON public.user_groups USING btree (name);


--
-- Name: idx_user_roles_role_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_roles_role_id ON public.user_roles USING btree (role_id);


--
-- Name: idx_user_roles_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_roles_user_id ON public.user_roles USING btree (user_id);


--
-- Name: idx_users_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_deleted_at ON public.users USING btree (deleted_at);


--
-- Name: idx_users_email; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_users_email ON public.users USING btree (email);


--
-- Name: idx_users_username; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_users_username ON public.users USING btree (username);


--
-- Name: idx_variables_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_variables_deleted_at ON public.variables USING btree (deleted_at);


--
-- Name: idx_variables_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_variables_project_id ON public.variables USING btree (project_id);


--
-- Name: hosts fk_projects_hosts; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.hosts
    ADD CONSTRAINT fk_projects_hosts FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;


--
-- Name: tasks fk_projects_tasks; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tasks
    ADD CONSTRAINT fk_projects_tasks FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;


--
-- Name: variables fk_projects_variables; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.variables
    ADD CONSTRAINT fk_projects_variables FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;


--
-- Name: role_bindings fk_role_bindings_role; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_bindings
    ADD CONSTRAINT fk_role_bindings_role FOREIGN KEY (role_id) REFERENCES public.roles(id);


--
-- Name: role_permissions fk_role_permissions_permission; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_permissions
    ADD CONSTRAINT fk_role_permissions_permission FOREIGN KEY (permission_id) REFERENCES public.permissions(id);


--
-- Name: role_permissions fk_role_permissions_role; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_permissions
    ADD CONSTRAINT fk_role_permissions_role FOREIGN KEY (role_id) REFERENCES public.roles(id);


--
-- Name: user_group_memberships fk_user_group_memberships_user; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_group_memberships
    ADD CONSTRAINT fk_user_group_memberships_user FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: user_group_memberships fk_user_group_memberships_user_group; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_group_memberships
    ADD CONSTRAINT fk_user_group_memberships_user_group FOREIGN KEY (user_group_id) REFERENCES public.user_groups(id);


--
-- Name: user_group_roles fk_user_group_roles_role; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_group_roles
    ADD CONSTRAINT fk_user_group_roles_role FOREIGN KEY (role_id) REFERENCES public.roles(id);


--
-- Name: user_group_roles fk_user_group_roles_user_group; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_group_roles
    ADD CONSTRAINT fk_user_group_roles_user_group FOREIGN KEY (user_group_id) REFERENCES public.user_groups(id);


--
-- Name: user_roles fk_user_roles_role; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT fk_user_roles_role FOREIGN KEY (role_id) REFERENCES public.roles(id);


--
-- Name: user_roles fk_user_roles_user; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT fk_user_roles_user FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: projects fk_users_projects; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.projects
    ADD CONSTRAINT fk_users_projects FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

