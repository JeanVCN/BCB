# Memória operacional do projeto BCB

> Documento destinado a modelos de inteligência artificial. Deve refletir o
> entendimento vigente do projeto e ser atualizado junto com as decisões do
> produto. Não substitui a documentação oficial nem o README.

## Metadados

- Projeto: Big Chat Brasil (BCB), desafio técnico Fullstack.
- Última consolidação: 2026-06-22.
- Estado atual: planejamento; implementação ainda não autorizada.
- Fonte inicial: documentos oficiais locais em `docs/` e definições do
  responsável pelo projeto.

## Protocolo de manutenção

Ao surgir uma nova definição:

1. atualizar este arquivo e `project-docs/product-decisions.md` na mesma
   alteração;
2. distinguir decisão confirmada, hipótese e pendência;
3. registrar a data e a justificativa;
4. nunca converter exemplos da documentação oficial em obrigações sem uma
   decisão explícita;
5. preservar decisões substituídas no histórico da documentação do projeto.

## Objetivo vigente

Construir uma aplicação Fullstack de chat empresarial que permita cadastrar e
autenticar empresas clientes, manter conversas com seus clientes finais, enviar
mensagens, controlar custos pré e pó-pagos, processar mensagens em fila e
acompanhar seu estado. A entrega deve favorecer correção, rastreabilidade,
integração completa e capacidade de explicação durante apresentação e live
coding.

## Vocabulário adotado

- **Empresa cliente**: titular da conta no BCB, identificada por CPF ou CNPJ.
- **Cliente final/destinatário**: pessoa com quem a empresa mantém uma
  conversa; inicialmente possui nome e telefone.
- **Mensagem enviada**: mensagem solicitada pela empresa cliente para um
  destinatário.

Essa nomenclatura deve ser preservada para evitar a ambiguidade do termo
"cliente" encontrada nos documentos oficiais.

## Decisões confirmadas

### Tecnologias e execução

- Backend: Go com Gin.
- Frontend: React com TypeScript.
- Persistência: PostgreSQL desde a primeira versão.
- Docker deve ser implementado para a execução do ambiente.
- Redis integrará o controle de concorrência das operações financeiras por
  meio de bloqueio distribuído, visando segurança no escalonamento horizontal.
- RabbitMQ não pertence ao escopo inicial. É uma evolução condicionada a
  tempo disponível.

### Documentação

- `docs/` é material oficial da empresa, foi reduzido aos documentos da vaga
  Fullstack, não deve ser corrigido e não será commitado.
- As referências ausentes a perfis Backend e Frontend não são lacunas do
  escopo deste projeto.
- A memória para IA e a documentação do projeto devem permanecer separadas do
  README e dos documentos oficiais.

### Contas, cadastro e segurança

- Deve existir cadastro inicial da empresa cliente para que o fluxo completo
  possa ser demonstrado.
- A empresa será identificada por CPF ou CNPJ, mas o documento sozinho não
  autentica o acesso.
- O acesso exigirá senha.
- A senha terá entre 15 e 128 caracteres e não poderá ser truncada
  silenciosamente.
- Por decisão do projeto, a senha deverá combinar letras, números e caractere
  especial. Essa composição é mais restritiva que as recomendações atuais de
  NIST/OWASP, que priorizam comprimento e bloqueio de senhas comprometidas; o
  trade-off deve ser explicado na apresentação.
- Senhas devem ser armazenadas somente como hash resistente e com salt.
- A autenticação deverá ter limitação de tentativas, e senhas comuns ou
  conhecidas como comprometidas deverão ser rejeitadas.
- Empresa cliente inativa não pode autenticar nem enviar mensagens.
- O onboarding adotado é o autocadastro: toda nova empresa cliente entra
  inativa e depende de ativação por um usuário administrador.
- O RBAC inicial possui os papéis `admin` e `client`.
- O administrador controla ativação/inativação, créditos, limites, planos e
  solicitações de mudança.
- Toda ação administrativa deve registrar ator, instante, operação e valores
  anteriores/novos quando aplicável.
- O primeiro administrador será criado por comando idempotente de bootstrap,
  com credenciais fornecidas por variáveis de ambiente e sem endpoint público.
- No autocadastro, o cliente informa o plano desejado. Na ativação, o
  administrador confirma o mesmo plano solicitado e define saldo ou limite
  inicial.
- Cadastro rejeitado permanece registrado para auditoria, não libera novo
  cadastro com o mesmo CPF/CNPJ e pode ser reconsiderado por administrador.
- Depois do autocadastro, a resposta informa que o cadastro foi recebido e
  aguarda aprovação, sem prometer prazo.
- A sessão inicial usa token com uma hora de validade, sem renovação automática;
  após expirar, é necessário autenticar novamente.
- CPF/CNPJ duplicado nunca cria uma segunda conta, inclusive após rejeição.
- Telefone de destinatário é único dentro de cada empresa; tentar abrir nova
  conversa para o mesmo telefone recupera a conversa existente.

### Conversas e mensagens

- Deve ser possível cadastrar uma nova conversa com, no mínimo, nome e telefone
  do cliente final.
- Não haverá integração inicial com provedor real de SMS ou WhatsApp.
- O envio obrigatório é interno e simulado: validar, cobrar, registrar,
  enfileirar, processar e atualizar o status.
- Não existe requisito oficial para recebimento por provedor externo. Mensagens
  recebidas usadas na demonstração podem vir de dados iniciais ou mecanismo
  interno a ser definido posteriormente.
- O canal SMS/WhatsApp pertence ao domínio documentado, embora a integração
  externa não pertença ao escopo inicial.
- O canal será selecionável desde a primeira versão.
- O ciclo inicial termina em `sent` ou `failed`.
- A estrutura deve permitir evolução posterior para `delivered` e `read`.
- Não haverá mensagem recebida/inbound nem status correspondente na primeira
  versão. `queued` e `processing` permanecem estados internos intermediários;
  os resultados terminais visíveis são `sent` e `failed`.

### Fila e processamento

- A primeira versão terá fila simples e worker.
- Mensagens urgentes têm precedência sobre normais, preservando FIFO dentro de
  cada prioridade.
- As responsabilidades da fila devem permanecer desacopladas o suficiente para
  futura substituição por um serviço dedicado/RabbitMQ.
- Retry faz parte do escopo essencial.
- Cada mensagem terá uma tentativa inicial e até três retries.
- Retries ocorrerão apenas para falhas transitórias, com backoff exponencial e
  jitter. As bases iniciais serão 1, 2 e 4 segundos, configuráveis e com teto de
  30 segundos.
- Falhas permanentes de validação ou negócio não entram em retry.
- Processamento, retry e efeitos financeiros devem ser idempotentes.

### Financeiro

- Os planos pré-pago e pós-pago existirão desde a primeira versão.
- Pré-pago exige saldo suficiente e débito antes do enfileiramento.
- Pós-pago deve armazenar separadamente o limite total e o limite consumido.
- A separação preserva a rastreabilidade quando o limite total é alterado no
  decorrer do período de cobrança.
- Administração financeira integra o escopo: créditos, limites, consultas,
  conversão de plano e histórico de transações.
- A conversão de plano somente pode ser efetivada quando não existir valor
  financeiro pendente: saldo pré-pago igual a zero ou consumo pós-pago igual a
  zero, conforme o plano atual.
- O cliente pode solicitar uma mudança de plano quando satisfizer essa condição.
- Cada cliente pode manter apenas uma solicitação de mudança pendente.
- A solicitação possui os estados `pending`, `approved`, `rejected` e
  `cancelled`.
- O cliente pode cancelar uma solicitação enquanto ela estiver pendente.
- Usuários administradores podem aprovar ou rejeitar solicitações e verão uma
  notificação interna, por contador e lista no painel administrativo.
- Não haverá inicialmente notificação por e-mail, push, WebSocket ou outro
  canal externo.
- A aprovação deve revalidar saldo/consumo e permissão no momento da operação,
  pois a situação pode ter mudado desde a solicitação.
- Solicitação, cancelamento, aprovação e rejeição devem ser auditáveis.
- Estorno em caso de falha faz parte do escopo essencial.
- O estorno ocorre somente depois de esgotadas as tentativas configuradas e a
  mensagem ser declarada definitivamente `failed`.
- O estorno deve ser idempotente, executado no máximo uma vez e registrado no
  histórico financeiro com referência à cobrança original.
- Redis será usado como coordenação distribuída; a consistência financeira
  também deverá ser garantida pela persistência transacional, sem depender
  exclusivamente do lock.

### Escopo posterior

Somente depois do essencial estar estável, avaliar:

- reset mensal do plano pós-pago;
- limite de conteúdo;
- ordenação configurável;
- paginação;
- RabbitMQ;
- estados posteriores a `sent`;
- recebimento simulado mais completo.

## Prioridade inicial de desenvolvimento

1. preparar ambiente Docker com PostgreSQL e Redis;
2. implementar cadastro e autenticação da empresa cliente;
3. implementar cadastro/listagem de conversas;
4. implementar os dois planos e administração financeira;
5. implementar envio, fila simples, worker, retry, estorno e status;
6. integrar a experiência completa no frontend;
7. adicionar testes, refinamentos e evoluções condicionadas ao tempo.

## Estratégia de versionamento

- O desenvolvimento individual ocorrerá diretamente na branch `main`.
- Não criar branches permanentes separadas para backend e frontend, evitando
  integração tardia e complexidade sem benefício para o desafio.
- Usar branches curtas somente para experimentos arriscados ou opcionais que
  possam ser descartados, como `experiment/rabbitmq`.
- Manter commits pequenos, coerentes, explicáveis e preferencialmente com build
  e testes verdes.
- Incluir migrations, testes e ajustes documentais junto da funcionalidade que
  os exige, sem criar commits artificiais por tipo de arquivo.
- Usar os prefixos `docs`, `chore`, `feat`, `fix`, `test` e `refactor` nas
  mensagens de commit.
- Evitar commits genéricos ou intermediários como `wip`, `ajustes` e `fix` sem
  contexto.
- O primeiro commit estabelece o marco documental anterior ao código, com a
  mensagem `docs: define product scope, decisions and contracts`.
- Fases previstas para o histórico: planejamento; fundação; cadastro/RBAC/auth;
  financeiro; conversas; mensagens/fila; mudança de plano; estabilização e
  entrega.
- Tags opcionais de marco: `planning-complete` após o planejamento e `v0.1.0`
  ou `mvp-complete` após o fluxo mínimo integrado.

## Pendências abertas

Não há pendência funcional bloqueante para iniciar a implementação. Novas
ambiguidades devem ser registradas antes de alterar os contratos aprovados.

## Onboarding e RBAC aprovados

- Adotar autocadastro da empresa em estado inativo.
- Permitir autenticação somente depois da ativação por usuário administrador.
- Usar inicialmente apenas os papéis `admin` e `client`.
- Reservar ao `admin` a ativação/inativação de contas, crédito, alteração de
  limite e conversão de plano.
- Registrar em auditoria o administrador, instante, operação e valores
  anteriores/novos.

Essa opção preserva o cadastro ponta a ponta, torna o estado ativo/inativo
demonstrável e evita que o administrador conheça ou distribua a senha do
cliente. A decisão foi confirmada em 2026-06-22.
