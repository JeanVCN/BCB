# Critérios de aceite do BCB

Este documento converte o escopo aprovado em comportamentos verificáveis. Ele
não define endpoints, banco de dados ou estrutura de código.

## 1. Ambiente e entrega

- [x] Todo o ambiente necessário pode ser iniciado com Docker.
- [ ] Backend, frontend, PostgreSQL e Redis conseguem se comunicar no ambiente
  documentado.
- [ ] Uma instalação limpa possui um caminho documentado para criar o primeiro
  administrador.
- [ ] Dados persistidos no PostgreSQL sobrevivem ao reinício dos serviços.
- [ ] A aplicação apresenta erros de configuração sem expor segredos.

## 2. Cadastro e ativação

- [ ] Uma empresa consegue se cadastrar como PF/CPF ou PJ/CNPJ.
- [ ] CPF/CNPJ duplicado não cria uma segunda conta.
- [ ] A senha aceita entre 15 e 128 caracteres e não é truncada.
- [ ] A senha exige letras, números e caractere especial, conforme decisão do
  projeto.
- [ ] Senhas conhecidas como comuns ou comprometidas são rejeitadas.
- [ ] A senha nunca é persistida ou registrada em texto puro.
- [ ] Uma empresa recém-cadastrada permanece inativa.
- [ ] Uma empresa inativa não consegue autenticar nem enviar mensagens.
- [ ] Um administrador consegue consultar cadastros aguardando ativação.
- [ ] Somente um administrador consegue ativar ou inativar uma empresa.
- [ ] Ativação e inativação registram ator, instante e mudança realizada.

## 3. Autenticação e autorização

- [ ] Uma empresa ativa autentica com documento e senha corretos.
- [ ] Credenciais inválidas retornam uma resposta que não revela qual campo
  estava incorreto.
- [ ] Tentativas de autenticação são limitadas progressivamente.
- [ ] Um cliente acessa somente seus próprios dados, conversas e mensagens.
- [ ] Um usuário `client` não executa operações exclusivas de `admin`.
- [ ] Uma sessão inválida ou expirada não acessa recursos protegidos.

## 4. Planos e administração financeira

- [ ] Uma empresa possui exatamente um plano vigente: pré-pago ou pós-pago.
- [ ] Administrador consegue adicionar crédito ao pré-pago.
- [ ] Administrador consegue alterar o limite total do pós-pago sem apagar seu
  consumo acumulado.
- [ ] Saldo, limite total, consumo e disponibilidade são apresentados de forma
  coerente com o plano.
- [ ] Toda alteração financeira cria histórico rastreável.
- [ ] Operações financeiras concorrentes não produzem saldo negativo, consumo
  perdido ou cobrança duplicada.
- [ ] Indisponibilidade do lock distribuído não permite prosseguir de forma
  insegura com uma mutação financeira.

## 5. Solicitação de mudança de plano

- [ ] Cliente pré-pago somente solicita mudança com saldo igual a zero.
- [ ] Cliente pós-pago somente solicita mudança com consumo igual a zero.
- [ ] Um cliente não cria duas solicitações pendentes.
- [ ] O cliente pode cancelar sua solicitação pendente.
- [ ] O painel administrativo exibe contador e lista de solicitações pendentes.
- [ ] Um administrador pode aprovar ou rejeitar uma solicitação.
- [ ] A aprovação revalida autorização e condição financeira.
- [ ] Uma aprovação inválida não altera o plano.
- [ ] `pending`, `approved`, `rejected` e `cancelled` são transições
  controladas e auditáveis.

## 6. Conversas

- [ ] Uma empresa ativa consegue cadastrar conversa com nome e telefone do
  cliente final.
- [ ] A empresa visualiza somente suas conversas.
- [ ] A lista apresenta destinatário e resumo da atividade mais recente.
- [ ] Selecionar uma conversa carrega seu histórico.
- [ ] Estados vazio, carregando e erro possuem apresentação apropriada.

## 7. Envio e financeiro da mensagem

- [ ] A empresa escolhe SMS ou WhatsApp e prioridade normal ou urgente.
- [ ] Mensagem normal custa R$ 0,25 e urgente custa R$ 0,50.
- [ ] Pré-pago sem saldo suficiente não é cobrado nem enfileirado.
- [ ] Pós-pago sem limite disponível não é consumido nem enfileirado.
- [ ] Uma mensagem válida registra cliente, conversa, destinatário, canal,
  conteúdo, prioridade, custo e instante.
- [ ] Cobrança, registro da mensagem e encaminhamento para processamento não
  deixam estado financeiro parcial observável.
- [ ] Repetir uma mesma solicitação identificada não gera mensagem ou cobrança
  duplicada.

## 8. Fila, worker, retry e estorno

- [ ] Mensagem aceita inicia como `queued` e passa por `processing`.
- [ ] O worker processa uma mensagem no máximo uma vez simultaneamente.
- [ ] A fila simples preserva a ordenação definida para a primeira versão.
- [ ] Uma falha transitória realiza no máximo três retries depois da tentativa
  inicial.
- [ ] Retries usam backoff exponencial com jitter e configuração testável.
- [ ] Falha permanente não é repetida.
- [ ] Sucesso termina em `sent` e não gera estorno.
- [ ] Esgotamento dos retries termina em `failed`.
- [ ] Uma mensagem definitivamente `failed` gera exatamente um estorno ligado
  à transação original.
- [ ] Retry, processamento e estorno permanecem idempotentes após reinício ou
  execução concorrente.

## 9. Frontend e integração

- [ ] O fluxo cadastro → ativação → login → conversa → envio pode ser
  demonstrado de ponta a ponta.
- [ ] O frontend apresenta loading, vazio, erro e sucesso nas operações
  principais.
- [ ] A lista e o histórico refletem o estado persistido no backend.
- [ ] O frontend acompanha `queued`, `processing`, `sent` e `failed` sem
  apresentar falso sucesso.
- [ ] Falhas de saldo, limite, validação, autorização e comunicação possuem
  mensagens compreensíveis.
- [ ] O layout funciona de maneira utilizável em mobile e desktop.
- [ ] Recarregar a página não perde dados persistidos e trata a sessão
  corretamente.

## 10. Auditoria e segurança operacional

- [ ] Operações administrativas e financeiras relevantes possuem auditoria.
- [ ] Logs não expõem senha, token ou outros segredos.
- [ ] Erros internos não devolvem detalhes sensíveis ao frontend.
- [ ] Datas e valores financeiros mantêm representação consistente entre as
  camadas.

## Fora do primeiro escopo

- Integração real com SMS ou WhatsApp.
- Mensagens inbound/recebidas de provedor externo.
- Estados `delivered` e `read`.
- RabbitMQ.
- E-mail, push ou WebSocket para notificações administrativas.
- Reset mensal automatizado, paginação e busca avançada.

## Definição de pronto desta fase

Os comportamentos deste documento estão modelados em `domain-model.md` e
`api-contracts.md`. Novas ambiguidades descobertas durante a implementação devem
ser registradas antes de alterar os contratos.
