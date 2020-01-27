FROM ubuntu:latest

ARG DB_PASSWORD
ENV DEBIAN_FRONTEND=noninteractive

# Install dependency package
RUN apt-get update && apt-get install gnupg gnupg2 wget sudo net-tools -y
# Add repository
# Import the repository signing key
RUN echo 'deb http://apt.postgresql.org/pub/repos/apt/ bionic-pgdg main' >> /etc/apt/sources.list.d/pgdg.list && \
    wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add -
RUN apt-get update && \
    apt-get install postgresql-9.6 -y

USER postgres

RUN /etc/init.d/postgresql start && \
    psql --variable password=${DB_PASSWORD} --command "CREATE USER pgadmin WITH SUPERUSER PASSWORD '${DB_PASSWORD}';" && \
    createdb --owner=pgadmin mydb

# Enable Remote Access
RUN echo "host all all 0.0.0.0/0 md5" >> /etc/postgresql/9.6/main/pg_hba.conf
RUN echo "listen_addresses='*'" >> /etc/postgresql/9.6/main/postgresql.conf
# RUN echo "shared_buffers=1GB" >> /etc/postgresql/9.6/main/postgresql.conf

EXPOSE 5432
CMD ["/usr/lib/postgresql/9.6/bin/postgres", \
     "-D", "/var/lib/postgresql/9.6/main", \
     "-c", "config_file=/etc/postgresql/9.6/main/postgresql.conf"]
