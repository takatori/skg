FROM solr:9.4.1

USER root

ADD solr.xml ./server/solr/solr.xml
ADD run_solr.sh ./run_solr.sh
RUN chown solr:solr run_solr.sh
RUN sed -i -e 's/\r$//' run_solr.sh
RUN chmod u+x run_solr.sh

ADD log4j2-config.xml ./log4j2-config.xml

USER solr

ENTRYPOINT "./run_solr.sh" 
