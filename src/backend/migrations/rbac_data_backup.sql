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

--
-- Data for Name: permissions; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.permissions (id, resource, verb, scope, description, created_at, updated_at) FROM stdin;
1	*	*	*	超级管理员权限	2025-06-07 15:44:52.6437+00	2025-06-07 15:44:52.6437+00
2	projects	create	*	创建项目权限	2025-06-07 15:44:52.644311+00	2025-06-07 15:44:52.644311+00
3	projects	read	*	查看项目权限	2025-06-07 15:44:52.644711+00	2025-06-07 15:44:52.644711+00
4	projects	update	*	更新项目权限	2025-06-07 15:44:52.645158+00	2025-06-07 15:44:52.645158+00
5	projects	delete	*	删除项目权限	2025-06-07 15:44:52.645548+00	2025-06-07 15:44:52.645548+00
6	projects	list	*	列出项目权限	2025-06-07 15:44:52.645988+00	2025-06-07 15:44:52.645988+00
7	users	create	*	创建用户权限	2025-06-07 15:44:52.646434+00	2025-06-07 15:44:52.646434+00
8	users	read	*	查看用户权限	2025-06-07 15:44:52.646829+00	2025-06-07 15:44:52.646829+00
9	users	update	*	更新用户权限	2025-06-07 15:44:52.647384+00	2025-06-07 15:44:52.647384+00
10	users	delete	*	删除用户权限	2025-06-07 15:44:52.647833+00	2025-06-07 15:44:52.647833+00
11	users	list	*	列出用户权限	2025-06-07 15:44:52.648307+00	2025-06-07 15:44:52.648307+00
12	roles	*	*	角色管理权限	2025-06-07 15:44:52.648721+00	2025-06-07 15:44:52.648721+00
13	groups	*	*	用户组管理权限	2025-06-07 15:44:52.649128+00	2025-06-07 15:44:52.649128+00
\.


--
-- Data for Name: roles; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.roles (id, name, description, is_system, created_at, updated_at, deleted_at) FROM stdin;
1	super-admin	超级管理员角色	t	2025-06-07 15:44:52.649852+00	2025-06-07 15:44:52.649852+00	\N
2	user	普通用户角色	t	2025-06-07 15:44:52.65117+00	2025-06-07 15:44:52.65117+00	\N
\.


--
-- Data for Name: role_permissions; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.role_permissions (id, role_id, permission_id, created_at) FROM stdin;
1	1	1	2025-06-07 15:44:52.650129+00
2	2	2	2025-06-07 15:44:52.651308+00
3	2	3	2025-06-07 15:44:52.651451+00
4	2	4	2025-06-07 15:44:52.651605+00
5	2	5	2025-06-07 15:44:52.651707+00
6	2	6	\N
\.


--
-- Data for Name: users; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.users (id, username, email, password, is_active, last_login, created_at, updated_at, deleted_at) FROM stdin;
1	admin	admin@example.com	$2a$10$6DStxiZuiSWAo3XKljeGyOXYE7.hMx.Pjt3xQFl21PWKphJCbDfzS	t	2025-06-07 15:58:22.130175+00	2025-06-07 15:44:52.707674+00	2025-06-07 15:58:22.130261+00	\N
2	testuser	test@example.com	$2a$10$xaMA5dTZT.yjQTpQJiXqPujQA/X45DY286I.3o/GQoVk6sKwBWRha	t	2025-06-07 16:09:52.162429+00	2025-06-07 15:48:52.602941+00	2025-06-07 16:09:52.162528+00	\N
\.


--
-- Data for Name: user_roles; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.user_roles (id, user_id, role_id, created_at) FROM stdin;
1	1	1	2025-06-07 15:44:52.709303+00
3	2	2	\N
\.


--
-- Name: permissions_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.permissions_id_seq', 13, true);


--
-- Name: role_permissions_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.role_permissions_id_seq', 6, true);


--
-- Name: roles_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.roles_id_seq', 2, true);


--
-- Name: user_roles_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.user_roles_id_seq', 3, true);


--
-- Name: users_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.users_id_seq', 2, true);


--
-- PostgreSQL database dump complete
--

