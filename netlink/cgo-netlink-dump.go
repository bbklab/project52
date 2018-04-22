package main

/*
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <unistd.h>
#include <asm/types.h>
#include <sys/socket.h>
#include <linux/netlink.h>
#include <linux/rtnetlink.h>
#include <netinet/in.h>
#include <linux/tcp.h>
#include <linux/sock_diag.h>
#include <linux/inet_diag.h>
#include <arpa/inet.h>
#include <pwd.h>

typedef struct {
	int established;
	int synsent;
	int synrecv;
	int finwait1;
	int finwait2;
	int timewait;
	int close;
	int closewait;
	int lastack;
	int listen;
	int closing;
} counter;

enum{
    TCP_ESTABLISHED = 1,
    TCP_SYN_SENT,
    TCP_SYN_RECV,
    TCP_FIN_WAIT1,
    TCP_FIN_WAIT2,
    TCP_TIME_WAIT,
    TCP_CLOSE,
    TCP_CLOSE_WAIT,
    TCP_LAST_ACK,
    TCP_LISTEN,
    TCP_CLOSING
};

#define TCPF_ALL 0xFFF
#define SOCKET_BUFFER_SIZE (getpagesize() < 8192L ? getpagesize() : 8192L)

int send_diag_msg(int sockfd){
    struct msghdr msg;
    struct nlmsghdr nlh;
    struct inet_diag_req_v2 conn_req;
    struct sockaddr_nl sa;
    struct iovec iov[4];
    int retval = 0;

    struct rtattr rta;

    memset(&msg, 0, sizeof(msg));
    memset(&sa, 0, sizeof(sa));
    memset(&nlh, 0, sizeof(nlh));
    memset(&conn_req, 0, sizeof(conn_req));

    sa.nl_family = AF_NETLINK;

    conn_req.sdiag_family = AF_INET;
    conn_req.sdiag_protocol = IPPROTO_TCP;

    conn_req.idiag_states = TCPF_ALL;

    conn_req.idiag_ext |= (1 << (INET_DIAG_INFO - 1));

    nlh.nlmsg_len = NLMSG_LENGTH(sizeof(conn_req));
    nlh.nlmsg_flags = NLM_F_DUMP | NLM_F_REQUEST;
    nlh.nlmsg_type = SOCK_DIAG_BY_FAMILY;
    iov[0].iov_base = (void*) &nlh;
    iov[0].iov_len = sizeof(nlh);
    iov[1].iov_base = (void*) &conn_req;
    iov[1].iov_len = sizeof(conn_req);

    msg.msg_name = (void*) &sa;
    msg.msg_namelen = sizeof(sa);
    msg.msg_iov = iov;
	msg.msg_iovlen = 2;

    retval = sendmsg(sockfd, &msg, 0);

    return retval;
}

void parse_diag_msg(struct inet_diag_msg *diag_msg, int rtalen, counter* c){
    struct rtattr *attr;
    struct tcp_info *tcpi;
    if(rtalen > 0){
        attr = (struct rtattr*) (diag_msg+1);
        while(RTA_OK(attr, rtalen)){
            if(attr->rta_type == INET_DIAG_INFO){
                tcpi = (struct tcp_info*) RTA_DATA(attr);
				if (tcpi->tcpi_state == TCP_ESTABLISHED ) {
					c->established +=1;
				} else if (tcpi->tcpi_state == TCP_SYN_SENT) {
					c->synsent +=1;
				} else if (tcpi->tcpi_state == TCP_SYN_RECV) {
					c->synrecv +=1;
				} else if (tcpi->tcpi_state == TCP_FIN_WAIT1) {
					c->finwait1+=1;
				} else if (tcpi->tcpi_state == TCP_FIN_WAIT2) {
					c->finwait2+=1;
				} else if (tcpi->tcpi_state == TCP_TIME_WAIT) {
					c->timewait+=1;
				} else if (tcpi->tcpi_state == TCP_CLOSE) {
					c->close+=1;
				} else if (tcpi->tcpi_state == TCP_CLOSE_WAIT) {
					c->closewait+=1;
				} else if (tcpi->tcpi_state == TCP_LAST_ACK) {
					c->lastack+=1;
				} else if (tcpi->tcpi_state == TCP_LISTEN) {
					c->listen +=1;
				} else if (tcpi->tcpi_state == TCP_CLOSING) {
					c->closing+=1;
				}
            }
            attr = RTA_NEXT(attr, rtalen);
        }
    }
}

void initcounter(counter* c) {
	c->established = 0;
	c->synsent = 0;
	c->synrecv = 0;
	c->finwait1 = 0;
	c->finwait2 = 0;
	c->timewait = 0;
	c->close = 0;
	c->closewait = 0;
	c->lastack = 0;
	c->listen = 0;
	c->closing = 0;
}

counter dump(){
    int nl_sock = 0, numbytes = 0, rtalen = 0;
    struct nlmsghdr *nlh;
    uint8_t recv_buf[SOCKET_BUFFER_SIZE];
    struct inet_diag_msg *diag_msg;
	counter c;
	initcounter(&c);

    if((nl_sock = socket(AF_NETLINK, SOCK_DGRAM, NETLINK_INET_DIAG)) == -1){
        perror("socket: ");
        return c;
    }

    if(send_diag_msg(nl_sock) < 0){
        perror("sendmsg: ");
		close(nl_sock);
        return c;
    }

    while(1){
        numbytes = recv(nl_sock, recv_buf, sizeof(recv_buf), 0);
        nlh = (struct nlmsghdr*) recv_buf;

        while(NLMSG_OK(nlh, numbytes)){
            if(nlh->nlmsg_type == NLMSG_DONE)
				goto outer;

            if(nlh->nlmsg_type == NLMSG_ERROR){
                fprintf(stderr, "Error in netlink message\n");
				close(nl_sock);
                return c;
            }

            diag_msg = (struct inet_diag_msg*) NLMSG_DATA(nlh);
            rtalen = nlh->nlmsg_len - NLMSG_LENGTH(sizeof(*diag_msg));
            parse_diag_msg(diag_msg, rtalen, &c);

            nlh = NLMSG_NEXT(nlh, numbytes);
        }
    }

outer:
	close(nl_sock);
    return c;
}
*/
import "C"
import (
	"fmt"
	"log"
)

func main() {
	c, err := C.dump()
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("%#v\n", c)
}
